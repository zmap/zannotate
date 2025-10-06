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

	"github.com/oschwald/maxminddb-golang/v2"
	log "github.com/sirupsen/logrus"
)

// This module provides IPInfo.io annotations for IP addresses using a local MaxMind DB file.

// ------------------------------------------------------------------------------------
// The MaxMind DB formats were pulled from IPInfo.io's API documentation on 07/29/2025.
// IPInfo provides data at various tiers of access: Lite, Core, and Plus.
// Since the MaxMindDB decode is best-effort, we'll just define the Plus format which includes all lower tiers.
// If a user has a Lite or Core DB file, the fields not present in those tiers will not appear in output.
// See https://ipinfo.io/products/plus for more info on field definitions.
// ------------------------------------------------------------------------------------

// IPInfoMMDBOutput includes both the Plus/Core/Lite IPInfo fields and their maxminddb tags. We'll convert this into a
// IPInfoModuleOutput for JSON output (converting string fields to appropriate types).
type IPInfoMMDBOutput struct {
	City              string `maxminddb:"city"`
	Region            string `maxminddb:"region"`
	RegionCode        string `maxminddb:"region_code"`
	Country           string `maxminddb:"country"`
	CountryCode       string `maxminddb:"country_code"`
	Continent         string `maxminddb:"continent"`
	ContinentCode     string `maxminddb:"continent_code"`
	Latitude          string `maxminddb:"latitude"`
	Longitude         string `maxminddb:"longitude"`
	Timezone          string `maxminddb:"timezone"`
	PostalCode        string `maxminddb:"postal_code"`
	GeonameID         string `maxminddb:"geoname_id"`  // GeoNames database identifier (if available).
	Radius            string `maxminddb:"radius"`      // Accuracy radius in kilometers (if available).
	GeoChanged        string `maxminddb:"geo_changed"` // Timestamp or flag indicating when the geolocation last changed (if available).
	ASN               string `maxminddb:"asn"`
	ASName            string `maxminddb:"as_name"`
	ASDomain          string `maxminddb:"as_domain"`
	ASType            string `maxminddb:"as_type"`
	ASChanged         string `maxminddb:"as_changed"`
	CarrierName       string `maxminddb:"carrier_name"` // Name of the mobile carrier (if available).
	MobileCountryCode string `maxminddb:"mcc"`
	MobileNetworkCode string `maxminddb:"mnc"`
	PrivacyName       string `maxminddb:"privacy_name"` // Specific name of the privacy or anonymization service detected (e.g., “NordVPN”).
	IsProxy           string `maxminddb:"is_proxy"`
	IsRelay           string `maxminddb:"is_relay"`     // Boolean flag indicating use of a general relay service
	IsTOR             string `maxminddb:"is_tor"`       // Whether the IP is a known TOR exit node.
	IsVPN             string `maxminddb:"is_vpn"`       // Flag indicating use of a VPN Service
	IsAnonymous       string `maxminddb:"is_anonymous"` // True if the IP is associated with VPN, proxy, Tor, or a relay service.
	IsAnycast         string `maxminddb:"is_anycast"`   // Whether the IP is using anycast routing.
	IsHosting         string `maxminddb:"is_hosting"`   // True if the IP address is an internet service hosting IP address
	IsMobile          string `maxminddb:"is_mobile"`    // True if the IP address is associated with a mobile network or carrier.
	IsSatellite       string `maxminddb:"is_satellite"` // True if the IP address is associated with a satellite connection
}

// IPInfoModuleOutput is the final output struct with appropriate types for JSON output
type IPInfoModuleOutput struct {
	City              string  `json:"city,omitempty"`
	Region            string  `json:"region,omitempty"`
	RegionCode        string  `json:"region_code,omitempty"`
	Country           string  `json:"country,omitempty"`
	CountryCode       string  `json:"country_code,omitempty"`
	Continent         string  `json:"continent,omitempty"`
	ContinentCode     string  `json:"continent_code,omitempty"`
	Latitude          float64 `json:"latitude,omitempty"`
	Longitude         float64 `json:"longitude,omitempty"`
	Timezone          string  `json:"timezone,omitempty"`
	PostalCode        string  `json:"postal_code,omitempty"`
	GeonameID         uint64  `json:"geoname_id,omitempty"`  // GeoNames database identifier (if available).
	Radius            uint64  `json:"radius,omitempty"`      // Accuracy radius in kilometers (if available).
	GeoChanged        string  `json:"geo_changed,omitempty"` // Timestamp or flag indicating when the geolocation last changed (if available).
	ASN               string  `json:"asn,omitempty"`
	ASName            string  `json:"as_name,omitempty"`
	ASDomain          string  `json:"as_domain,omitempty"`
	ASType            string  `json:"as_type,omitempty"`
	ASChanged         string  `json:"as_changed,omitempty"`
	CarrierName       string  `json:"carrier_name,omitempty"` // Name of the mobile carrier (if available).
	MobileCountryCode string  `json:"mobile_country_code,omitempty"`
	MobileNetworkCode string  `json:"mobile_network_code,omitempty"`
	PrivacyName       string  `json:"privacy_name,omitempty"` // Specific name of the privacy or anonymization service detected (e.g., “NordVPN”).
	IsProxy           *bool   `json:"is_proxy,omitempty"`
	IsRelay           *bool   `json:"is_relay,omitempty"`     // Boolean flag indicating use of a general relay service
	IsTOR             *bool   `json:"is_tor,omitempty"`       // Whether the IP is a known TOR exit node.
	IsVPN             *bool   `json:"is_vpn,omitempty"`       // Flag indicating use of a VPN Service
	IsAnonymous       *bool   `json:"is_anonymous,omitempty"` // True if the IP is associated with VPN, proxy, Tor, or a relay service.
	IsAnycast         *bool   `json:"is_anycast,omitempty"`   // Whether the IP is using anycast routing.
	IsHosting         *bool   `json:"is_hosting,omitempty"`   // True if the IP address is an internet service hosting IP address
	IsMobile          *bool   `json:"is_mobile,omitempty"`    // True if the IP address is associated with a mobile network or carrier.
	IsSatellite       *bool   `json:"is_satellite,omitempty"` // True if the IP address is associated with a satellite connection
}

func (in *IPInfoMMDBOutput) ToModuleOutput() *IPInfoModuleOutput {
	out := &IPInfoModuleOutput{
		City:              in.City,
		Region:            in.Region,
		RegionCode:        in.RegionCode,
		Country:           in.Country,
		CountryCode:       in.CountryCode,
		Continent:         in.Continent,
		ContinentCode:     in.ContinentCode,
		Timezone:          in.Timezone,
		PostalCode:        in.PostalCode,
		GeoChanged:        in.GeoChanged,
		ASN:               in.ASN,
		ASName:            in.ASName,
		ASDomain:          in.ASDomain,
		ASType:            in.ASType,
		ASChanged:         in.ASChanged,
		CarrierName:       in.CarrierName,
		MobileCountryCode: in.MobileCountryCode,
		MobileNetworkCode: in.MobileNetworkCode,
		PrivacyName:       in.PrivacyName,
	}
	// Convert string fields to appropriate types
	var err error
	if out.Latitude, err = strconv.ParseFloat(in.Latitude, 64); err != nil {
		out.Latitude = 0
	}
	if out.Longitude, err = strconv.ParseFloat(in.Longitude, 64); err != nil {
		out.Longitude = 0
	}
	if out.GeonameID, err = strconv.ParseUint(in.GeonameID, 10, 64); err != nil {
		out.GeonameID = 0
	}
	if out.Radius, err = strconv.ParseUint(in.Radius, 10, 64); err != nil {
		out.Radius = 0
	}
	var temp bool
	if temp, err = strconv.ParseBool(in.IsProxy); err == nil {
		out.IsProxy = &temp
	}
	if temp, err = strconv.ParseBool(in.IsRelay); err == nil {
		out.IsRelay = &temp
	}
	if temp, err = strconv.ParseBool(in.IsTOR); err == nil {
		out.IsTOR = &temp
	}
	if temp, err = strconv.ParseBool(in.IsVPN); err == nil {
		out.IsVPN = &temp
	}
	if temp, err = strconv.ParseBool(in.IsAnonymous); err == nil {
		out.IsAnonymous = &temp
	}
	if temp, err = strconv.ParseBool(in.IsAnycast); err == nil {
		out.IsAnycast = &temp
	}
	if temp, err = strconv.ParseBool(in.IsHosting); err == nil {
		out.IsHosting = &temp
	}
	if temp, err = strconv.ParseBool(in.IsMobile); err == nil {
		out.IsMobile = &temp
	}
	if temp, err = strconv.ParseBool(in.IsSatellite); err == nil {
		out.IsSatellite = &temp
	}
	return out
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
	// On a quick benchmark of 1M IPs using a local DB file on a M2 Macbook Air, 1 thread vs. 10 threads were about the same speed, annotating about 212k IPs/second.
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
	var out *IPInfoMMDBOutput
	if err = a.Factory.db.Lookup(ip).Decode(&out); err != nil {
		log.Debugf("error decoding IP %s in IPInfo database: %v", ip.String(), err)
	}
	if out == nil {
		return nil // no data found for this IP
	}
	return out.ToModuleOutput() // convert from the full-string struct to a typed struct
}

func (a *IPInfoAnnotator) Close() error {
	return nil
}

func init() {
	s := new(IPInfoAnnotatorFactory)
	RegisterAnnotator(s)
}
