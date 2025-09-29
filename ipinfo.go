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
	log "github.com/sirupsen/logrus"
)

// This module provides IPInfo.io annotations for IP addresses using a local MaxMind DB file.

// ------------------------------------------------------------------------------------
// MaxMind DB format definitions and conversion functions
// The MaxMind DB formats were pulled from IPInfo.io's API documentation on 07/29/2025.
// IPInfo provides data at various tiers of access: Lite, Core, and Plus.
// Since the MaxMindDB decode is best-effort, we'll just define the Plus format which includes all lower tiers.
// If a user has a Lite or Core DB file, the fields not present in those tiers will not appear in output.
// See https://ipinfo.io/products/plus for more info on field definitions.
// ------------------------------------------------------------------------------------

// IPInfoOutput includes both the Plus/Core/Lite IPInfo fields and their maxminddb tags, as well as the JSON tags for ZAnnotate output.
type IPInfoOutput struct {
	City              string  `maxminddb:"city" json:"city,omitempty"`
	Region            string  `maxminddb:"region" json:"region,omitempty"`
	RegionCode        string  `maxminddb:"region_code" json:"region_code,omitempty"`
	Country           string  `maxminddb:"country" json:"country,omitempty"`
	CountryCode       string  `maxminddb:"country_code" json:"country_code,omitempty"`
	Continent         string  `maxminddb:"continent" json:"continent,omitempty"`
	ContinentCode     string  `maxminddb:"continent_code" json:"continent_code,omitempty"`
	Latitude          float64 `maxminddb:"latitude" json:"latitude,omitempty"`
	Longitude         float64 `maxminddb:"longitude" json:"longitude,omitempty"`
	Timezone          string  `maxminddb:"timezone" json:"timezone,omitempty"`
	PostalCode        string  `maxminddb:"postal_code" json:"postal_code,omitempty"`
	GeonameID         string  `maxminddb:"geoname_id" json:"geoname_id,omitempty"`   // GeoNames database identifier (if available).
	Radius            int     `maxminddb:"radius" json:"radius,omitempty"`           // Accuracy radius in kilometers (if available).
	GeoChanged        string  `maxminddb:"geo_changed" json:"geo_changed,omitempty"` // Timestamp or flag indicating when the geolocation last changed (if available).
	ASN               string  `maxminddb:"asn" json:"asn,omitempty"`
	ASName            string  `maxminddb:"as_name" json:"as_name,omitempty"`
	ASDomain          string  `maxminddb:"as_domain" json:"as_domain,omitempty"`
	ASType            string  `maxminddb:"as_type" json:"as_type,omitempty"`
	ASChanged         string  `maxminddb:"as_changed" json:"as_changed,omitempty"`
	CarrierName       string  `maxminddb:"carrier_name" json:"carrier_name,omitempty"` // Name of the mobile carrier (if available).
	MobileCountryCode string  `maxminddb:"mcc" json:"mobile_country_code,omitempty"`
	MobileNetworkCode string  `maxminddb:"mnc" json:"mobile_network_code,omitempty"`
	PrivacyName       string  `maxminddb:"privacy_name" json:"privacy_name,omitempty"` // Specific name of the privacy or anonymization service detected (e.g., “NordVPN”).
	IsProxy           *bool   `maxminddb:"is_proxy" json:"is_proxy,omitempty"`
	IsRelay           *bool   `maxminddb:"is_relay" json:"is_relay,omitempty"`         // Boolean flag indicating use of a general relay service
	IsTOR             *bool   `maxminddb:"is_tor" json:"is_tor,omitempty"`             // Whether the IP is a known TOR exit node.
	IsVPN             *bool   `maxminddb:"is_vpn" json:"is_vpn,omitempty"`             // Flag indicating use of a VPN Service
	IsAnonymous       *bool   `maxminddb:"is_anonymous" json:"is_anonymous,omitempty"` // True if the IP is associated with VPN, proxy, Tor, or a relay service.
	IsAnycast         *bool   `maxminddb:"is_anycast" json:"is_anycast,omitempty"`     // Whether the IP is using anycast routing.
	IsHosting         *bool   `maxminddb:"is_hosting" json:"is_hosting,omitempty"`     // True if the IP address is an internet service hosting IP address
	IsMobile          *bool   `maxminddb:"is_mobile" json:"is_mobile,omitempty"`       // True if the IP address is associated with a mobile network or carrier.
	IsSatellite       *bool   `maxminddb:"is_satellite" json:"is_satellite,omitempty"` // True if the IP address is associated with a satellite connection
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
	flags.BoolVar(&a.Enabled, "ipinfo", false, "annotate with IPInfo.io data using a local MaxMind DB file")
	flags.StringVar(&a.DatabaseFilePath, "ipinfo-database", "", "path to MaxMind DB data file for IPInfo.io annotation")
	// TODO Phillip - performance test for optimal thread count
	flags.IntVar(&a.Threads, "ipinfo-threads", 1, "how many ipinfo annotator threads")
}

// IPInfo Annotator (Per-Worker)

func (a *IPInfoAnnotator) Initialize() error {
	return nil
}

func (a *IPInfoAnnotator) GetFieldName() string {
	return "ipinfo"
}

func (a *IPInfoAnnotator) Annotate(inputIP net.IP) interface{} {
	ip, err := netip.ParseAddr(inputIP.String())
	if err != nil {
		return nil // not a valid IP address, nothing to be done
	}
	// Decode the IP address using the MaxMind DB reader
	var out *IPInfoOutput
	if err = a.Factory.db.Lookup(ip).Decode(&out); out != nil {
		return out
	}
	if err != nil {
		log.Debugf("error decoding IP %s in IPInfo database: %v", ip.String(), err)
	}
	return nil
}

func (a *IPInfoAnnotator) Close() error {
	return nil
}

func init() {
	s := new(IPInfoAnnotatorFactory)
	RegisterAnnotator(s)
}
