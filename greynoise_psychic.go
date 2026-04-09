/*
 * ZAnnotate Copyright 2026 Regents of the University of Michigan
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
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang/v2"
	log "github.com/sirupsen/logrus"
)

type GreyNoiseAnnotatorFactory struct {
	BasePluginConf
	DBPath      string // path to the .mmdb path
	greynoiseDB *maxminddb.Reader
}

// GreyNoise Annotator Factory (Global)

func (a *GreyNoiseAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v GreyNoiseAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *GreyNoiseAnnotatorFactory) Initialize(_ *GlobalConf) error {
	if len(a.DBPath) == 0 {
		return errors.New("greynoise database path is required when greynoise annotator is enabled, use --greynoise-database")
	}
	data, err := os.ReadFile(a.DBPath) // ensure DB is read in-memory
	if err != nil {
		return fmt.Errorf("unable to read greynoise database at %s: %w", a.DBPath, err)
	}
	a.greynoiseDB, err = maxminddb.OpenBytes(data)
	if err != nil {
		return fmt.Errorf("unable to open greynoise database at %s: %w", a.DBPath, err)
	}
	return nil
}

func (a *GreyNoiseAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *GreyNoiseAnnotatorFactory) Close() error {
	if err := a.greynoiseDB.Close(); err != nil {
		return fmt.Errorf("unable to close greynoise database at %s: %w", a.DBPath, err)
	}
	return nil
}

func (a *GreyNoiseAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *GreyNoiseAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	// Reverse DNS Lookup
	flags.BoolVar(&a.Enabled, "greynoise", false, "greynoise psychic data intelligence")
	flags.StringVar(&a.DBPath, "greynoise-database", "", "path to greynoise psychic .mmdb file")
	flags.IntVar(&a.Threads, "greynoise-threads", 2, "how many enrichment threads to use")
}

// GreyNoiseAnnotator (Per-Worker)
type GreyNoiseAnnotator struct {
	Factory *GreyNoiseAnnotatorFactory
	Id      int
}

func (a *GreyNoiseAnnotator) Initialize() (err error) {
	return nil
}

func (a *GreyNoiseAnnotator) GetFieldName() string {
	return "greynoise"
}

// Annotate performs a reverse DNS lookup for the given IP address and returns the results.
// If an error occurs or a lookup fails, it returns nil
func (a *GreyNoiseAnnotator) Annotate(ip net.IP) interface{} {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		log.Debugf("unable to convert IP %s to address", ip)
		return nil
	}
	addr = addr.Unmap()
	var result any
	err := a.Factory.greynoiseDB.Lookup(addr).Decode(&result)
	if err != nil {
		log.Debugf("unable to annotate IP (%s): %v", addr, err)
		return nil
	}
	return result
}

func (a *GreyNoiseAnnotator) Close() error {
	return nil
}

func init() {
	s := new(GreyNoiseAnnotatorFactory)
	RegisterAnnotator(s)
}
