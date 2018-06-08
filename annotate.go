package zannotate

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
	Ip net.IP
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
	retv.Out = make(map[string]interface{})
	return retv
}

func AnnotateRead(conf *GlobalConf, path string, in chan<- inProcessIP) {
	log.Debug("read thread started")
	var f *os.File
	if path == "" || path == "-" {
		log.Debug("reading input from stdin")
		f = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			log.Fatal("unable to open input file:", err.Error())
		}
		log.Debug("reading input from ", path)
	}
	r := bufio.NewReader(f)
	line, err := r.ReadString('\n')
	// read IPs out of JSON input
	for err == nil {
		l := strings.TrimSuffix(line, "\n")
		if conf.InputFileType == "json" {
			in <- jsonToInProcess(l, conf.JSONIPFieldName, conf.JSONAnnotationFieldName)
		} else {
			in <- ipToInProcess(l)
		}
		line, err = r.ReadString('\n')
	}
	if err != nil && err != io.EOF {
		log.Fatal("input unable to read file", err)
	}
	close(in)
	log.Debug("read thread finished")
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

func AnnotateWorker(a *Annotator, r inProcessIP, wg *sync.WaitGroup, i int) {
	name := a.GetFieldName()
	log.Debug("annotate worker (", name, ") ", i, " started")
	if err := a.Initialize(); err != nil {
		log.Fatal("error initializing annotate worker: ", err)
	}
	for inProcess := range in {
		inProcess.Out[name] = a.Annotate(inProcess.Ip)
	}
	wg.Done()
}

func DoAnnotation(conf *GlobalConf) {
	outChan := make(chan string)
	inChan := make(chan inProcessIP)

	var outputWG sync.WaitGroup
	outputWG.Add(1)

	//startTime := time.Now().Format(time.RFC3339)
	go AnnotateRead(conf, conf.InputFilePath, inChan)
	go AnnotateWrite(conf.OutputFilePath, outChan, &outputWG)

	var annotateWG sync.WaitGroup
	annotateWG.Add(conf.Threads)
	for i := 0; i < conf.Threads; i++ {
		go AnnotateWorker(conf, inChan, outChan, &annotateWG, i)
	}
	annotateWG.Wait()
	close(outChan)
	outputWG.Wait()
	//endTime := time.Now().Format(time.RFC3339)
}
