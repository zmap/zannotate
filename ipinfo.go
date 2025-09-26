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
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/oschwald/maxminddb-golang/v2"
	log "github.com/sirupsen/logrus"
)

// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-
// IPInfo.io CSV format
// network,country,country_code,continent,continent_code,asn,as_name,as_domain
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-

type IPInfoOutput struct {
	Country       string `json:"country,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	Continent     string `json:"continent,omitempty"`
	ContinentCode string `json:"continent_code,omitempty"`
	ASN           int    `json:"asn,omitempty"`
	ASName        string `json:"as_name,omitempty"`
	ASDomain      string `json:"as_domain,omitempty"`
}

type IPInfoAnnotatorFactory struct {
	BasePluginConf
	DatabaseFilePath string
	db               *maxminddb.Reader // MMDB Database Reader is thread-safe
}

type IPInfoAnnotator struct {
	Factory *IPInfoAnnotatorFactory
	Id      int
}

// IPInfo Annotator Factory (Global)

func (a *IPInfoAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v IPInfoAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *IPInfoAnnotatorFactory) Initialize(conf *GlobalConf) (err error) {
	if len(a.DatabaseFilePath) == 0 {
		return errors.New("ipinfo database file path is required")
	}
	if a.db, err = maxminddb.Open(a.DatabaseFilePath); err != nil {
		return fmt.Errorf("error opening IPInfo database reader: %w", err)
	}
	// verify the MaxMind DB is not corrupted
	if err = a.db.Verify(); err != nil {
		return fmt.Errorf("error occured while trying to validate the MaxMind DB file: %w", err)
	}
	return nil
}

func (a *IPInfoAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *IPInfoAnnotatorFactory) Close() error {
	if err := a.db.Close(); err != nil {
		return fmt.Errorf("error closing IPInfo database reader: %w", err)
	}
	return nil
}

func (a *IPInfoAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *IPInfoAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "ipinfo", false, "annotate with IPInfo.io data")
	flags.StringVar(&a.DatabaseFilePath, "ipinfo-database", "", "path to IPInfo.io MMDB data file")
	flags.IntVar(&a.Threads, "ipinfo-threads", 1, "how many ipinfo annotator threads")
}

// IPInfo Annotator (Per-Worker)

func (a *IPInfoAnnotator) Initialize() error {
	return nil
}

func (a *IPInfoAnnotator) GetFieldName() string {
	return "ipinfo"
}

// IPInfo Lite Tier Record
type liteRecord struct {
	Country       string `maxminddb:"country"`
	CountryCode   string `maxminddb:"country_code"`
	Continent     string `maxminddb:"continent"`
	ContinentCode string `maxminddb:"continent_code"`
	ASN           string `maxminddb:"asn"`
	ASName        string `maxminddb:"as_name"`
	ASDomain      string `maxminddb:"as_domain"`
}

// Convert the liteRecord to the output format
func (lite *liteRecord) toIPInfoOutput() *IPInfoOutput {
	if lite == nil {
		return nil
	}
	out := &IPInfoOutput{
		Country:       lite.Country,
		CountryCode:   lite.CountryCode,
		Continent:     lite.Continent,
		ContinentCode: lite.ContinentCode,
		ASName:        lite.ASName,
		ASDomain:      lite.ASDomain,
	}
	var err error
	const AsPrefix = "AS"
	trimmedPrefix, _ := strings.CutPrefix(lite.ASN, AsPrefix)
	out.ASN, err = strconv.Atoi(trimmedPrefix)
	if err != nil {
		out.ASN = 0 // omit-empty will not output this field
	}
	return out
}

func (a *IPInfoAnnotator) Annotate(inputIP net.IP) interface{} {
	ip, err := netip.ParseAddr(inputIP.String())
	if err != nil {
		return nil // not a valid IP address, nothing to be done
	}

	// IPInfo has multiple tiers of access. To deal with this and to be resilient to DB changes,
	// we'll attempt to decode into a custom struct first, then fallback to a generic any type.
	var out *liteRecord
	if _ = a.Factory.db.Lookup(ip).Decode(&out); out != nil {
		return out.toIPInfoOutput() // Convert to our standard output format
	}
	// Fallback to using any since we don't know the exact entry structure
	// TODO Phillip
	// Tradeoff between being resilient to DB changes and having a stable output format
	// So we might not want to do the following in the long-term
	var record any
	if err := a.Factory.db.Lookup(ip).Decode(&record); err != nil {
		log.Debugf("error looking up IP %s in IPInfo database: %v", ip.String(), err)
		return nil
	}
	return record
}

func (a *IPInfoAnnotator) Close() error {
	return nil
}

func init() {
	s := new(IPInfoAnnotatorFactory)
	RegisterAnnotator(s)
}
