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
	"io/ioutil"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/oschwald/geoip2-golang"
)

type GeoIP2Conf struct {
	Path       string
	Mode       string
	Language   string
	RawInclude string
	// what data to include
	IncludeCity               bool
	IncludeCountry            bool
	IncludeContinent          bool
	IncludePostal             bool
	IncludeLatLong            bool
	IncludeTraits             bool
	IncludeSubdivisions       bool
	IncludeRepresentedCountry bool
	IncludeRegisteredCountry  bool
}

type GeoIP2Output struct {
	City struct {
		Name      string `json:"name"`
		GeoNameId uint   `json:"id"`
	} `json:"city,omitempty"`
	Country struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		GeoNameId uint   `json:"id"`
	} `json:"country,omitempty"`
	Continent struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		GeoNameId uint   `json:"id"`
	} `json:"continent,omitempty"`
	Postal struct {
		Code string `json:"code"`
	} `json:"postal,omitempty"`
	LatLong struct {
		AccuracyRadius uint16  `json:"accuracy_radius"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		MetroCode      uint    `json:"metro_code"`
		TimeZone       string  `json:"time_zone"`
	} `json:"latlong,omitempty"`
	RepresentedCountry struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		GeoNameId uint   `json:"id"`
	} `json:"represented_country,omitempty"`
	RegisteredCountry struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		GeoNameId uint   `json:"id"`
	} `json:"represented_country,omitempty"`
	Traits struct {
		IsAnonymousProxy    bool `json:"is_anonymous_proxy"`
		IsSatelliteProvider bool `json:"is_satellite_provider"`
	} `json:"metadata,omitempty"`
}

func GeoIP2ParseRawIncludeString(conf *GeoIP2Conf) {
	if conf.RawInclude == "*" {
		log.Debug("will include all geoip fields")
		conf.IncludeCity = true
		conf.IncludeCountry = true
		conf.IncludeContinent = true
		conf.IncludePostal = true
		conf.IncludeLatLong = true
		conf.IncludeTraits = true
		conf.IncludeSubdivisions = true
		conf.IncludeRegisteredCountry = true
		conf.IncludeRepresentedCountry = true
	} else {
		log.Debug("will include GeoIP fields: ", conf.RawInclude)
		for _, s := range strings.Split(conf.RawInclude, ",") {
			ps := strings.Trim(s, " ")
			switch ps {
			case "city":
				conf.IncludeCity = true
			case "country":
				conf.IncludeCountry = true
			case "continent":
				conf.IncludeContinent = true
			case "latlong":
				conf.IncludeLatLong = true
			case "postal":
				conf.IncludePostal = true
			case "traits":
				conf.IncludeTraits = true
			case "subdivisions":
				conf.IncludeSubdivisions = true
			case "registered_country":
				conf.IncludeRegisteredCountry = true
			case "represented_country":
				conf.IncludeRepresentedCountry = true
			default:
				log.Fatal("Invalid GeoIP2 field: ", ps)
			}
		}
	}
}

func GeoIP2FillStruct(in *geoip2.City, conf *GeoIP2Conf) *GeoIP2Output {
	var out GeoIP2Output
	if conf.IncludeCity == true {
		out.City.Name = in.City.Names[conf.Language]
		out.City.GeoNameId = in.City.GeoNameID
	}
	if conf.IncludeCountry == true {
		out.Country.Name = in.Country.Names[conf.Language]
		out.Country.GeoNameId = in.Country.GeoNameID
		out.Country.Code = in.Country.IsoCode
	}
	if conf.IncludeContinent == true {
		out.Continent.Name = in.Continent.Names[conf.Language]
		out.Continent.GeoNameId = in.Continent.GeoNameID
		out.Continent.Code = in.Continent.Code
	}
	if conf.IncludeLatLong == true {
		out.LatLong.AccuracyRadius = in.Location.AccuracyRadius
		out.LatLong.Latitude = in.Location.Latitude
		out.LatLong.Longitude = in.Location.Longitude
		out.LatLong.MetroCode = in.Location.MetroCode
		out.LatLong.TimeZone = in.Location.TimeZone
	}
	if conf.IncludePostal == true {
		out.Postal.Code = in.Postal.Code
	}
	if conf.IncludeTraits == true {
		out.Traits.IsAnonymousProxy = in.Traits.IsAnonymousProxy
		out.Traits.IsSatelliteProvider = in.Traits.IsSatelliteProvider
	}
	if conf.IncludeRegisteredCountry == true {
		out.RegisteredCountry.Name = in.RegisteredCountry.Names[conf.Language]
		out.RegisteredCountry.GeoNameId = in.RegisteredCountry.GeoNameID
		out.RegisteredCountry.Code = in.RegisteredCountry.IsoCode
	}
	if conf.IncludeRepresentedCountry == true {
		out.RepresentedCountry.Name = in.RepresentedCountry.Names[conf.Language]
		out.RepresentedCountry.GeoNameId = in.RepresentedCountry.GeoNameID
		out.RepresentedCountry.Code = in.RepresentedCountry.IsoCode
	}
	return &out
}

func GeoIP2Open(conf *GeoIP2Conf) *geoip2.Reader {
	if conf.Mode == "memory" {
		bytes, err := ioutil.ReadFile(conf.Path)
		if err != nil {
			log.Fatal("unable to open maxmind geoIP2 database (memory): ", err)
		}
		db, err := geoip2.FromBytes(bytes)
		if err != nil {
			log.Fatal("unable to parse maxmind geoIP2 database: ", err)
		}
		return db
	} else if conf.Mode == "mmap" {
		db, err := geoip2.Open(conf.Path)
		if err != nil {
			log.Fatal("unable to load maxmind geoIP2 database: ", err)
		}
		return db
	}
	log.Fatal("invalid GeoIP mode")
	return nil
}
