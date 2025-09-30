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

// Note - MaxMind's GeoLite databases used be known as GeoIP2.

type GeoLiteASNOutput struct {
	ASN    uint   `json:"asn,omitempty"`
	ASNOrg string `json:"org,omitempty"`
}

type GeoLiteASNAnnotatorFactory struct {
	BasePluginConf
	Path string
}

type GeoLiteASNAnnotator struct {
	Factory *GeoLiteASNAnnotatorFactory
	Reader  *geoip2.Reader
	Id      int
}

func (fact *GeoLiteASNAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&fact.Enabled, "geoasn", false, "annotate with Maxmind GeoLite ASN data")
	flags.StringVar(&fact.Path, "geoasn-database", "", "path to Maxmind ASN database")
	flags.IntVar(&fact.Threads, "geoasn-threads", 5, "how many geoASN processing threads to use")
}

func (fact *GeoLiteASNAnnotatorFactory) IsEnabled() bool {
	return fact.Enabled
}

func (fact *GeoLiteASNAnnotatorFactory) GetWorkers() int {
	return fact.Threads
}

func (fact *GeoLiteASNAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v GeoLiteASNAnnotator
	v.Factory = fact
	v.Id = i
	return &v
}

func (fact *GeoLiteASNAnnotatorFactory) Initialize(_ *GlobalConf) error {
	if fact.Path == "" {
		log.Fatal("no GeoIP ASN database provided")
	}
	log.Info("Will add ASNs using ", fact.Path)
	return nil
}

func (fact *GeoLiteASNAnnotatorFactory) Close() error {
	return nil
}

func (anno *GeoLiteASNAnnotator) Initialize() error {
	bytes, err := os.ReadFile(anno.Factory.Path)
	if err != nil {
		return fmt.Errorf("unable to open maxmind GeoLite ASN database (memory): %w", err)
	}
	db, err := geoip2.FromBytes(bytes)
	if err != nil {
		return fmt.Errorf("unable to parse maxmind GeoLite ASN database: %w", err)
	}
	anno.Reader = db
	return nil
}

func (anno *GeoLiteASNAnnotator) GetFieldName() string {
	return "geoasn"
}

func (anno *GeoLiteASNAnnotator) Annotate(ip net.IP) interface{} {
	record, err := anno.Reader.ASN(ip)
	if err != nil {
		return &GeoLiteASNOutput{}
	}
	return &GeoLiteASNOutput{
		ASN:    record.AutonomousSystemNumber,
		ASNOrg: record.AutonomousSystemOrganization,
	}
}

func (anno *GeoLiteASNAnnotator) Close() error {
	return anno.Reader.Close()
}

func init() {
	f := new(GeoLiteASNAnnotatorFactory)
	RegisterAnnotator(f)
}
