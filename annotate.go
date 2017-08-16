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
	"log"
	"net"
	"os"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type GlobalConf struct {
	InputFilePath    string
	InputFileType    string
	OutputFilePath   string
	MetadataFilePath string
	LogFilePath      string
	Verbosity        int
	Threads          int

	GeoIP2     bool
	GeoIP2Conf GeoIP2Conf
}

type Result struct {
	ip     string       `json:"ip,omitempty"`
	geoip2 GeoIP2Output `json:"geoip2,omitempty"`
}

func reader(path string, in chan<- string) {
	var f *os.File
	if path == "" || path == "-" {
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(path)
		if err != nil {
			log.Fatal("unable to open input file:", err.Error())
		}
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		in <- s.Text()
	}
	if err := s.Err(); err != nil {
		log.Fatal("input unable to read file", err)
	}
}

func AnnotateWorker(conf *GlobalConf, in <-chan string, out chan<- string, wg *sync.WaitGroup) {
	// not entirely sure if this geolocation library is thread safe
	// so for now we're just going to open the MaxMind database in every thraed
	var geoIP2db *geoip2.Reader
	if conf.GeoIP2 {
		geoIP2db = GeoIP2Open(&conf.GeoIP2Conf)
		defer geoIP2db.Close()
	}
	for line := range in {
		ip := net.ParseIP(line)
		if ip == nil {
			log.Fatal("invalid IP received: ", line)
		}
		var out Result
		if conf.GeoIP2 == true {
			record, err := geoIP2db.City(ip)
			if err != nil {
				log.Fatal(err)
			}
			out.geoip2 = GeoIP2FillStruct(record, &conf.GeoIP2Conf)
		}
	}
	wg.Done()
}

func DoAnnotation(conf GlobalConf) {
	outChan := make(chan string)
	inChan := make(chan string)
	var lookupWG sync.WaitGroup
	lookupWG.Add(conf.Threads)
	//startTime := time.Now().Format(time.RFC3339)
	for i := 0; i < conf.Threads; i++ {
		go AnnotateWorker(&conf, inChan, outChan, &lookupWG)
	}
	lookupWG.Wait()

}
