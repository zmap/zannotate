/*
 * ZAnnotate Copyright 2017 Regents of the University of Michigan
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

package zannotate

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"sync"

	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
)

type GlobalConf struct {
	InputFilePath           string
	InputFileType           string
	OutputFilePath          string
	MetadataFilePath        string
	LogFilePath             string
	Verbosity               int
	Threads                 int
	JSONIPFieldName         string
	JSONAnnotationFieldName string

	GeoIP2     bool
	GeoIP2Conf GeoIP2Conf

	Routing     bool
	RoutingConf RoutingConf
}

type Result struct {
	Ip      string         `json:"ip,omitempty"`
	GeoIP2  *GeoIP2Output  `json:"geoip2,omitempty"`
	Routing *RoutingOutput `json:"routing,omitempty"`
}

func AnnotateRead(path string, in chan<- string) {
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
	s := bufio.NewScanner(f)
	for s.Scan() {
		in <- s.Text()
	}
	if err := s.Err(); err != nil {
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

func AnnotateWorker(conf *GlobalConf, in <-chan string, out chan<- string,
	wg *sync.WaitGroup, i int) {
	log.Debug("annotate worker ", i, " started")
	// not entirely sure if this geolocation library is thread safe
	// so for now we're just going to open the MaxMind database in every thraed
	var geoIP2db *geoip2.Reader
	if conf.GeoIP2 {
		geoIP2db = GeoIP2Open(&conf.GeoIP2Conf)
		defer geoIP2db.Close()
	}
	log.Debug("annotate worker ", i, " initialization finished")
	for line := range in {
		// all lookup operations performed off of IP, which we parse into
		// depending on the configuration type
		var ip net.IP
		// JSON use only, but must be accessible throughout the loop
		var jsonMap map[string]interface{}
		if conf.InputFileType == "json" {
			var inParsed interface{}
			if err := json.Unmarshal([]byte(line), &inParsed); err != nil {
				log.Fatal("unable to parse json: ", line)
			}
			jsonMap = inParsed.(map[string]interface{})
			if val, ok := jsonMap[conf.JSONIPFieldName]; ok {
				if valS, ok := val.(string); ok {
					ip = net.ParseIP(valS)
				} else {
					log.Fatal("ip is not a string in JSON for ", line)
				}
			} else {
				log.Fatal("unable to find IP address field in ", line)
			}
			if _, ok := jsonMap[conf.JSONAnnotationFieldName]; ok {
				log.Fatal("input record already contains annotation key ", line)
			}
		} else {
			ip = net.ParseIP(line)
		}
		if ip == nil {
			log.Fatal("invalid IP received: ", line)
		}
		var res Result
		if conf.GeoIP2 == true {
			record, err := geoIP2db.City(ip)
			if err != nil {
				log.Fatal(err)
			}
			res.GeoIP2 = GeoIP2FillStruct(record, &conf.GeoIP2Conf)
		}
		if conf.Routing {
			res.Routing = RoutingFillStruct(ip, &conf.RoutingConf)
		}
		if conf.InputFileType == "json" {
			jsonMap[conf.JSONAnnotationFieldName] = res
			jsonRes, err := json.Marshal(jsonMap)
			if err != nil {
				log.Fatal("Unable to marshal JSON result", err)
			}
			out <- string(jsonRes)

		} else {
			res.Ip = ip.String()
			jsonRes, err := json.Marshal(res)
			if err != nil {
				log.Fatal("Unable to marshal JSON result", err)
			}
			out <- string(jsonRes)
		}
	}
	wg.Done()
	log.Debug("annotate worker ", i, " finished")
}

func DoAnnotation(conf *GlobalConf) {
	outChan := make(chan string)
	inChan := make(chan string)

	var outputWG sync.WaitGroup
	outputWG.Add(1)

	//startTime := time.Now().Format(time.RFC3339)
	go AnnotateRead(conf.InputFilePath, inChan)
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
