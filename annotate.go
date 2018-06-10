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
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// struct that is populated by the input reader and passed between types of worker threads
type inProcessIP struct {
	Out map[string]interface{}
	Ip  net.IP
}

func jsonToInProcess(line string, ipFieldName string, annotationFieldName string) inProcessIP {
	var inParsed interface{}
	var retv inProcessIP
	if err := json.Unmarshal([]byte(line), &inParsed); err != nil {
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
	return retv
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
	line, err := r.ReadString('\n')
	// read IPs out of JSON input
	for err == nil {
		in <- line
		line, err = r.ReadString('\n')
	}
	if err != nil && err != io.EOF {
		log.Fatal("input unable to read file", err)
	}
	close(in)
	log.Debug("read thread finished")
}

// multiple workers that decode raw lines from AnnotateRead from JSON/CSV into native golang objects
func AnnotateInputDecode(conf *GlobalConf, inChan <-chan string, outChan chan<- inProcessIP, wg *sync.WaitGroup, i int) {
	for line := range inChan {
		l := strings.TrimSuffix(line, "\n")
		if conf.InputFileType == "json" {
			outChan <- jsonToInProcess(l, conf.JSONIPFieldName, conf.JSONAnnotationFieldName)
		} else {
			outChan <- ipToInProcess(l)
		}
	}
	wg.Done()
}

func AnnotateOutputEncode(conf *GlobalConf, inChan <-chan inProcessIP, outChan chan<- string, wg *sync.WaitGroup, i int) {
	for rec := range inChan {
		jsonRes, err := json.Marshal(rec.Out)
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
		defer f.Close()
	}
	for n := range out {
		f.WriteString(n + "\n")
	}
	wg.Done()
	log.Debug("write thread finished")
}

func AnnotateWorker(a Annotator, inChan <-chan inProcessIP, outChan chan<- inProcessIP, wg *sync.WaitGroup, i int) {
	name := a.GetFieldName()
	log.Debug("annotate worker (", name, ") ", i, " started")
	if err := a.Initialize(); err != nil {
		log.Fatal("error initializing annotate worker: ", err)
	}
	for inProcess := range inChan {
		inProcess.Out[name] = a.Annotate(inProcess.Ip)
		outChan <- inProcess
	}
	wg.Done()
}

func DoAnnotation(conf *GlobalConf) {
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
				go AnnotateWorker(annotator.MakeAnnotator(i), lastChannel, nextChannel, &annotateWG, i)
				decodeWG.Add(1)
			}
			lastChannel = nextChannel
			annotateWaitGroups = append(annotateWaitGroups, &annotateWG)
			annotateChannels = append(annotateChannels, lastChannel)
			nextChannel = make(chan inProcessIP)
		}
	}
	// encode raw data
	var encodeWG sync.WaitGroup
	encodedOut := make(chan string)
	for i := 0; i < conf.OutputEncodeThreads; i++ {
		go AnnotateOutputEncode(conf, lastChannel, encodedOut, &encodeWG, i)
		encodeWG.Add(1)
	}
	var writeWG sync.WaitGroup
	go AnnotateWrite(conf.OutputFilePath, encodedOut, &writeWG)
	writeWG.Add(1)
	// all workers started. close out everything in a safe order
	// inRaw: we don't need to wait on this because it'll close its own channel when it finishes
	// until that channel is closed, none of the decoder threads will finish. So, just wait for them
	decodeWG.Wait()
	close(inDecoded)
	// wait on all of the different types of annotation workers
	for i, wg := range annotateWaitGroups {
		wg.Wait()
		close(annotateChannels[i])
	}
	// wait for the encoders
	encodeWG.Wait()
	close(encodedOut)
	// wait on writing to file
	writeWG.Wait()
	//endTime := time.Now().Format(time.RFC3339)
}
