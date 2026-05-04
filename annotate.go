package zannotate

/*
 * ZAnnotate Copyright 2018 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"slices"
	"time"

	"io"
	"net"
	"os"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// struct that is populated by the input reader and passed between types of worker threads
type inProcessIP struct {
	Out map[string]interface{}
	Ip  net.IP
}

func jsonToInProcess(line string, ipFieldName string,
	annotationFieldName string) inProcessIP {
	var inParsed interface{}
	var retv inProcessIP
	if err := jsoniter.Unmarshal([]byte(line), &inParsed); err != nil {
		log.Fatal("unable to parse input json record: ", line)
	}
	jsonMap := inParsed.(map[string]interface{})
	if val, ok := jsonMap[ipFieldName]; ok {
		if valS, ok := val.(string); ok {
			retv.Ip = net.ParseIP(valS)
		} else {
			log.Fatal("ip is not a string in JSON for ", line)
		}
	} else {
		log.Fatal("unable to find IP address field in ", line)
	}
	if _, ok := jsonMap[annotationFieldName]; ok {
		log.Fatal("input record already contains annotation key ", line)
	}
	retv.Out = jsonMap
	return retv
}

func ipToInProcess(line string) inProcessIP {
	var retv inProcessIP
	retv.Ip = net.ParseIP(line)
	if retv.Ip == nil {
		log.Fatal("invalid input IP address: ", line)
	}
	retv.Out = make(map[string]interface{})
	retv.Out["ip"] = retv.Ip
	return retv
}

func csvToInProcess(record []string, headers []string, ipFieldName, annotationFieldName string) inProcessIP {
	var retv inProcessIP

	if len(record) != len(headers) {
		log.Fatalf("csv record length (%d) does not match header length (%d) for record: %v", len(record), len(headers), record)
	}

	outMap := make(map[string]interface{}, len(headers))
	for i, header := range headers {
		outMap[header] = record[i]
	}

	foundIPField := false
	for i, header := range headers {
		switch header {
		case ipFieldName:
			foundIPField = true
			retv.Ip = net.ParseIP(record[i])
			if retv.Ip == nil {
				log.Fatalf("unable to parse IP at column '%d': %s", i, record[i])
			}
		case annotationFieldName:
			log.Fatalf("csv headers already contains annotation key '%v'", annotationFieldName)
		default:
			outMap[header] = record[i]
		}
	}
	if !foundIPField {
		log.Fatalf("unable to find IP address field with IP field name %s in CSV record: %v", ipFieldName, record)
	}

	retv.Out = outMap
	return retv
}

func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)
		for val := range in {
			out1 <- val
			out2 <- val
		}
	}()

	return out1, out2
}

// single worker that reads from file and queues raw lines
func AnnotateRead(conf *GlobalConf, path string, in chan<- string) {
	log.Debug("read thread started")
	var f *os.File
	if path == "" || path == "-" {
		log.Debug("reading input from stdin")
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(path)
		if err != nil {
			log.Fatal("unable to open input file:", err.Error())
		}
		log.Debug("reading input from ", path)
	}
	r := bufio.NewReader(f)
	if conf.InputFileType == "csv" {
		// Need to extract the header
		header, err := r.ReadString('\n')
		if err != nil {
			log.Fatal("unable to read the CSV headers", err.Error())
		}
		csvReader := csv.NewReader(strings.NewReader(header))
		fields, err := csvReader.Read()
		if err != nil {
			log.Fatal("unable to parse CSV headers: ", err.Error())
		}
		conf.csvHeaders = fields
	}
	// read IPs out of input
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			in <- line
		}
		if err != nil {
			if err != io.EOF {
				log.Fatal("input unable to read file", err)
			}
			break
		}
	}
	close(in)
	log.Debug("read thread finished")
}

// multiple workers that decode raw lines from AnnotateRead
// from JSON/CSV into native golang objects
func AnnotateInputDecode(conf *GlobalConf, inChan <-chan string,
	outChan chan<- inProcessIP, wg *sync.WaitGroup, i int) {
	for line := range inChan {
		l := strings.TrimSuffix(line, "\n")
		switch conf.InputFileType {
		case "json":
			val := jsonToInProcess(l, conf.InputIPFieldName, conf.OutputAnnotationFieldName)
			if conf.OutputAnnotationFieldName != "" {
				val.Out[conf.OutputAnnotationFieldName] = make(map[string]interface{})
			}
			outChan <- val
		case "csv":
			r := csv.NewReader(strings.NewReader(l))
			row, err := r.Read()
			if err != nil {
				log.Errorf("failed to parse CSV line (%s): %v", l, err)
				continue
			}
			val := csvToInProcess(row, conf.csvHeaders, conf.InputIPFieldName, conf.OutputAnnotationFieldName)
			if conf.OutputAnnotationFieldName != "" {
				val.Out[conf.OutputAnnotationFieldName] = make(map[string]interface{})
			}
			outChan <- val
		default:
			outChan <- ipToInProcess(l)

		}
	}
	log.Debug("decode thread ", i, " done")
	wg.Done()
}

func AnnotateOutputEncode(conf *GlobalConf, inChan <-chan inProcessIP,
	outChan chan<- string, wg *sync.WaitGroup, i int) {
	//
	for rec := range inChan {
		jsonRes, err := jsoniter.Marshal(rec.Out)
		if err != nil {
			log.Fatal("Unable to marshal JSON result", err)
		}
		outChan <- string(jsonRes)
	}
	wg.Done()
}

func AnnotateWrite(path string, out <-chan string, wg *sync.WaitGroup) {
	log.Debug("write thread started")
	var f *os.File
	if path == "" || path == "-" {
		f = os.Stdout
	} else {
		var err error
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal("unable to open output file:", err.Error())
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Errorf("unable to close output file: %v", err)
			}
		}()
	}
	for n := range out {
		if _, err := f.WriteString(n + "\n"); err != nil {
			log.Fatalf("unable to write to output file: %v", err)
		}
	}
	wg.Done()
	log.Debug("write thread finished")
}

func AnnotateWorker(conf *GlobalConf, a Annotator, inChan <-chan inProcessIP,
	outChan chan<- inProcessIP, fieldName string, wg *sync.WaitGroup, i int) {
	name := a.GetFieldName()
	log.Debug("annotate worker (", name, ") ", i, " started")
	log.Debug("annotate worker (", name, ") ", i, " to use fieldname ", name)
	if err := a.Initialize(); err != nil {
		log.Fatal("error initializing annotate worker: ", err)
	}
	for inProcess := range inChan {
		if fieldName != "" && slices.Contains([]string{"json", "csv"}, conf.InputFileType) {
			p := inProcess.Out[fieldName].(map[string]interface{})
			p[name] = a.Annotate(inProcess.Ip)
		} else {
			inProcess.Out[name] = a.Annotate(inProcess.Ip)
		}
		outChan <- inProcess
	}
	wg.Done()
}

func PerSecondUpdateWorker(filePath string, outChan <-chan string, wg *sync.WaitGroup) {
	log.Debug("PerSecondUpdateWorker started")
	defer wg.Done()
	f := os.Stderr
	if filePath != "-" && filePath != "" {
		var err error
		f, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("unable to open per-second log file: %s", err.Error())
		}
		defer f.Close()
	}
	startTime := time.Now()
	ticker := time.NewTicker(time.Second)
	ipsAnnotated := 0
	for {
		select {
		case <-ticker.C:
			// Print per-second output summary
			timeSinceStart := time.Since(startTime)
			s := fmt.Sprintf("%02dh:%02dm:%02ds; %d ips annotated; %.02f ips/sec\n",
				int(timeSinceStart.Hours()),
				int(timeSinceStart.Minutes())%60,
				int(timeSinceStart.Seconds())%60,
				ipsAnnotated,
				float64(ipsAnnotated)/timeSinceStart.Seconds())
			_, err := f.WriteString(s)
			if err != nil {
				log.Fatalf("unable to write to log file: %v", err)
			}
		case _, ok := <-outChan:
			if !ok {
				timeSinceStart := time.Since(startTime)
				s := fmt.Sprintf("%02dh:%02dm:%02ds; Scan Complete; %d ips annotated; %.02f ips/sec\n",
					int(timeSinceStart.Hours()),
					int(timeSinceStart.Minutes())%60,
					int(timeSinceStart.Seconds())%60,
					ipsAnnotated,
					float64(ipsAnnotated)/timeSinceStart.Seconds())
				_, err := f.WriteString(s)
				if err != nil {
					log.Fatalf("unable to write to log file: %v", err)
				}
				return
			}
			ipsAnnotated++
		}
	}

}

func DoAnnotation(conf *GlobalConf) {
	// let each enabled annotator do their global initialization
	// before we ask them to generate threa Annotators
	for _, annotator := range Annotators {
		if annotator.IsEnabled() {
			if err := annotator.Initialize(conf); err != nil {
				log.Fatal("Annotation unable to initialize: ", err)
			}
		}
	}
	// several types of channels/subprocesses
	// [read from file](1) -> [decode input data](n,d=3) -> [annotator 1](n)
	//    -> [annotator 2](n) -> ... -> [annotator n](n) -> [encode output data](n,d=3)
	//    -> [write to file](1)
	inRaw := make(chan string)
	inDecoded := make(chan inProcessIP)
	// read input file
	go AnnotateRead(conf, conf.InputFilePath, inRaw)

	// decode input data
	var decodeWG sync.WaitGroup
	for i := 0; i < conf.InputDecodeThreads; i++ {
		go AnnotateInputDecode(conf, inRaw, inDecoded, &decodeWG, i)
		decodeWG.Add(1)
	}
	// spawn threads for each type of annotator
	lastChannel := inDecoded
	nextChannel := make(chan inProcessIP)
	var annotateWaitGroups []*sync.WaitGroup
	var annotateChannels []chan inProcessIP
	for _, annotator := range Annotators {
		if annotator.IsEnabled() {
			var annotateWG sync.WaitGroup
			for i := 0; i < annotator.GetWorkers(); i++ {
				go AnnotateWorker(conf, annotator.MakeAnnotator(i), lastChannel, nextChannel,
					conf.OutputAnnotationFieldName, &annotateWG, i)
				annotateWG.Add(1)
			}
			lastChannel = nextChannel
			annotateWaitGroups = append(annotateWaitGroups, &annotateWG)
			annotateChannels = append(annotateChannels, lastChannel)
			nextChannel = make(chan inProcessIP)
		}
	}
	// encode raw data
	var encodeWG sync.WaitGroup
	pipedEncodedOut := make(chan string)
	for i := 0; i < conf.OutputEncodeThreads; i++ {
		go AnnotateOutputEncode(conf, lastChannel, pipedEncodedOut, &encodeWG, i)
		encodeWG.Add(1)
	}
	var writeWG sync.WaitGroup
	encodedOut, updatesOut := Tee[string](pipedEncodedOut)
	go AnnotateWrite(conf.OutputFilePath, encodedOut, &writeWG)
	writeWG.Add(1)
	go PerSecondUpdateWorker(conf.LogFilePath, updatesOut, &writeWG)
	writeWG.Add(1)
	// all workers started. close out everything in a safe order
	// inRaw: we don't need to wait on this because it'll close its own channel
	// when it finishes until that channel is closed, none of the decoder threads
	// will finish. So, just wait for them.
	decodeWG.Wait()
	close(inDecoded)
	// wait on all of the different types of annotation workers
	for i, wg := range annotateWaitGroups {
		wg.Wait()
		close(annotateChannels[i])
	}
	// wait for the encoders
	encodeWG.Wait()
	close(pipedEncodedOut)
	// wait on writing to file
	writeWG.Wait()
	//endTime := time.Now().Format(time.RFC3339)
}
