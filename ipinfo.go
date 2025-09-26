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

	"github.com/oschwald/maxminddb-golang/v2"
)

// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-
// IPInfo.io CSV format
// network,country,country_code,continent,continent_code,asn,as_name,as_domain
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-

type IPInfoOutput struct {
	Network       net.IPNet `json:"network"`
	Country       string    `json:"country"`
	CountryCode   string    `json:"country_code"`
	Continent     string    `json:"continent"`
	ContinentCode string    `json:"continent_code"`
	ASN           int       `json:"asn"`
	ASName        string    `json:"as_name"`
	ASDomain      string    `json:"as_domain"`
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

// TODO Phillip
// Optimizations
// Check out Custom High-Performance Unmarshalling
// https://pkg.go.dev/github.com/oschwald/maxminddb-golang/v2#section-readme
func (a *IPInfoAnnotator) Annotate(inputIP net.IP) interface{} {
	var ip netip.Addr // MaxMind DB Reader requires a netip.IP object
	if ipv4IP := inputIP.To4(); ipv4IP != nil {
		ip = netip.AddrFrom4([4]byte(ipv4IP))
	} else if ipv6IP := inputIP.To16(); ipv6IP != nil {
		ip = netip.AddrFrom16([16]byte(ipv6IP))
	}
	if !ip.IsValid() {
		return nil // not a valid IP address, nothing to be done
	}
	var record any
	_ = a.Factory.db.Lookup(ip).Decode(&record)

	return record
}

func (a *IPInfoAnnotator) Close() error {
	return nil
}

func init() {
	s := new(IPInfoAnnotatorFactory)
	RegisterAnnotator(s)
}
