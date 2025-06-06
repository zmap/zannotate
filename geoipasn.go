/*
 * ZAnnotate Copyright 2025 Regents of the University of Michigan
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
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
)

type GeoIPASNOutput struct {
	ASN    uint   `json:"asn,omitempty"`
	ASNOrg string `json:"org,omitempty"`
}

type GeoIPASNAnnotatorFactory struct {
	BasePluginConf
	Path string
	Mode string
}

type GeoIPASNAnnotator struct {
	Factory *GeoIPASNAnnotatorFactory
	Reader  *geoip2.Reader
	Id      int
}

func (fact *GeoIPASNAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&fact.Enabled, "geoasn", false, "annotate with Maxmind Geolite ASN data")
	flags.StringVar(&fact.Path, "geoasn-database", "", "path to Maxmind ASN database")
	flags.StringVar(&fact.Mode, "geoasn-mode", "mmap", "how to open database: mmap or memory")
	flags.IntVar(&fact.Threads, "geoasn-threads", 5, "how many geoASN processing threads to use")
}

func (fact *GeoIPASNAnnotatorFactory) IsEnabled() bool {
	return fact.Enabled
}

func (fact *GeoIPASNAnnotatorFactory) GetWorkers() int {
	return fact.Threads
}

func (fact *GeoIPASNAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v GeoIPASNAnnotator
	v.Factory = fact
	v.Id = i
	return &v
}

func (fact *GeoIPASNAnnotatorFactory) Initialize(conf *GlobalConf) error {
	if fact.Path == "" {
		log.Fatal("no GeoIP ASN database provided")
	}
	log.Info("Will add ASNs using ", fact.Path)
	return nil
}

func (fact *GeoIPASNAnnotatorFactory) Close() error {
	return nil
}

func (anno *GeoIPASNAnnotator) Initialize() error {
	switch anno.Factory.Mode {
	case "memory":
		bytes, err := os.ReadFile(anno.Factory.Path)
		if err != nil {
			return fmt.Errorf("unable to open maxmind geoIP ASN database (memory): %w", err)
		}
		db, err := geoip2.FromBytes(bytes)
		if err != nil {
			return fmt.Errorf("unable to parse maxmind geoIP ASN database: %w", err)
		}
		anno.Reader = db
	case "mmap":
		db, err := geoip2.Open(anno.Factory.Path)
		if err != nil {
			return fmt.Errorf("unable to load maxmind geoIP ASN database: %w", err)
		}
		anno.Reader = db
	default:
		return fmt.Errorf("unrecognized geoIP ASN mode: %s", anno.Factory.Mode)
	}
	return nil
}

func (anno *GeoIPASNAnnotator) GetFieldName() string {
	return "geoasn"
}

func (anno *GeoIPASNAnnotator) Annotate(ip net.IP) interface{} {
	record, err := anno.Reader.ASN(ip)
	if err != nil {
		return &GeoIPASNOutput{}
	}
	return &GeoIPASNOutput{
		ASN:    record.AutonomousSystemNumber,
		ASNOrg: record.AutonomousSystemOrganization,
	}
}

func (anno *GeoIPASNAnnotator) Close() error {
	return anno.Reader.Close()
}

func init() {
	f := new(GeoIPASNAnnotatorFactory)
	RegisterAnnotator(f)
}
