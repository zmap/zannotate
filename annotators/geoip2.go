/*
 * ZAnnotate Copyright 2018 Regents of the University of Michigan
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
	"io/ioutil"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
)

type GeoIP2City struct {
	Name      string `json:"name"`
	GeoNameId uint   `json:"id"`
}

type GeoIP2Country struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	GeoNameId uint   `json:"id"`
}

type GeoIP2Postal struct {
	Code string `json:"code"`
}

type GeoIP2LatLong struct {
	AccuracyRadius uint16  `json:"accuracy_radius"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	MetroCode      uint    `json:"metro_code"`
	TimeZone       string  `json:"time_zone"`
}

type GeoIP2Traits struct {
	IsAnonymousProxy    bool `json:"is_anonymous_proxy"`
	IsSatelliteProvider bool `json:"is_satellite_provider"`
}

type GeoIP2Output struct {
	City               *GeoIP2City    `json:"city,omitempty"`
	Country            *GeoIP2Country `json:"country,omitempty"`
	Continent          *GeoIP2Country `json:"continent,omitempty"`
	Postal             *GeoIP2Postal  `json:"postal,omitempty"`
	LatLong            *GeoIP2LatLong `json:"latlong,omitempty"`
	RepresentedCountry *GeoIP2Country `json:"represented_country,omitempty"`
	RegisteredCountry  *GeoIP2Country `json:"represented_country,omitempty"`
	Traits             *GeoIP2Traits  `json:"metadata,omitempty"`
}

type GeoIP2AnnotatorFactory struct {
	BasePluginConf
	Path       string
	Mode       string
	Language   string
	RawInclude string

	Conf   *GlobalConf
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

type GeoIP2Annotator struct {
	Factory *GeoIP2AnnotatorFactory
	Reader *geoip2.Reader
	Id int
}

// GeoIP2 Annotator Factory (Global)
func (a *GeoIP2AnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "geoip2", false, "annotate with Maxmind GeoIP2 data")
	flags.StringVar(&a.Path, "geoip2-database", "",
		"path to MaxMind GeoIP2 database")
	flags.StringVar(&a.Mode, "geoip2-mode", "mmap",
		"how to open database: mmap or memory")
	flags.StringVar(&a.Language, "geoip2-language", "en",
		"how to open database: mmap or memory")
	flags.StringVar(&a.RawInclude, "geoip2-fields", "*",
		"city, continent, country, location, postal, registered_country, subdivisions, traits")
	flags.IntVar(&a.Threads, "geoip2-threads", 5, "how many geoIP processing threads to use")
}

func (a *GeoIP2AnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *GeoIP2AnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v GeoIP2Annotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *GeoIP2AnnotatorFactory) Initialize(conf *GlobalConf) error {
	if a.Path == "" {
		log.Fatal("no GeoIP2 database provided")
	}
	log.Info("will add geoip2 using ", a.Path)
	if a.RawInclude == "*" {
		log.Debug("will include all geoip fields")
		a.IncludeCity = true
		a.IncludeCountry = true
		a.IncludeContinent = true
		a.IncludePostal = true
		a.IncludeLatLong = true
		a.IncludeTraits = true
		a.IncludeSubdivisions = true
		a.IncludeRegisteredCountry = true
		a.IncludeRepresentedCountry = true
	} else {
		log.Debug("will include GeoIP fields: ", a.RawInclude)
		for _, s := range strings.Split(a.RawInclude, ",") {
			ps := strings.Trim(s, " ")
			switch ps {
			case "city":
				a.IncludeCity = true
			case "country":
				a.IncludeCountry = true
			case "continent":
				a.IncludeContinent = true
			case "latlong":
				a.IncludeLatLong = true
			case "postal":
				a.IncludePostal = true
			case "traits":
				a.IncludeTraits = true
			case "subdivisions":
				a.IncludeSubdivisions = true
			case "registered_country":
				a.IncludeRegisteredCountry = true
			case "represented_country":
				a.IncludeRepresentedCountry = true
			default:
				return fmt.Errorf("Invalid GeoIP2 field: ", ps)
			}
		}
	}
	return nil
}

func (a *GeoIP2AnnotatorFactory) Close() error {
	return nil
}

// GeoIP2 Annotator (Per-Worker)

func (a *GeoIP2Annotator) Initialize() error {
	if a.Factory.Mode == "memory" {
		bytes, err := ioutil.ReadFile(a.Factory.Path)
		if err != nil {
			log.Fatal("unable to open maxmind geoIP2 database (memory): ", err)
		}
		db, err := geoip2.FromBytes(bytes)
		if err != nil {
			log.Fatal("unable to parse maxmind geoIP2 database: ", err)
		}
		a.Reader = db
	} else if a.Factory.Mode == "mmap" {
		db, err := geoip2.Open(a.Factory.Path)
		if err != nil {
			log.Fatal("unable to load maxmind geoIP2 database: ", err)
		}
		a.Reader = db
	} else {
		log.Fatal("invalid GeoIP mode")
	}
	defer a.Reader.Close()
	return nil
}


func (a *GeoIP2Annotator) GeoIP2FillStruct(in *geoip2.City) *GeoIP2Output {
	language := a.Factory.Language
	var out GeoIP2Output
	if a.Factory.IncludeCity == true {
		var city GeoIP2City
		out.City = &city
		out.City.Name = in.City.Names[language]
		out.City.GeoNameId = in.City.GeoNameID
	}
	if a.Factory.IncludeCountry == true {
		var country GeoIP2Country
		out.Country = &country
		out.Country.Name = in.Country.Names[language]
		out.Country.GeoNameId = in.Country.GeoNameID
		out.Country.Code = in.Country.IsoCode
	}
	if a.Factory.IncludeContinent == true {
		var country GeoIP2Country
		out.Continent = &country
		out.Continent.Name = in.Continent.Names[language]
		out.Continent.GeoNameId = in.Continent.GeoNameID
		out.Continent.Code = in.Continent.Code
	}
	if a.Factory.IncludeLatLong == true {
		var latlong GeoIP2LatLong
		out.LatLong = &latlong
		out.LatLong.AccuracyRadius = in.Location.AccuracyRadius
		out.LatLong.Latitude = in.Location.Latitude
		out.LatLong.Longitude = in.Location.Longitude
		out.LatLong.MetroCode = in.Location.MetroCode
		out.LatLong.TimeZone = in.Location.TimeZone
	}
	if a.Factory.IncludePostal == true {
		var postal GeoIP2Postal
		out.Postal = &postal
		out.Postal.Code = in.Postal.Code
	}
	if a.Factory.IncludeTraits == true {
		var traits GeoIP2Traits
		out.Traits = &traits
		out.Traits.IsAnonymousProxy = in.Traits.IsAnonymousProxy
		out.Traits.IsSatelliteProvider = in.Traits.IsSatelliteProvider
	}
	if a.Factory.IncludeRegisteredCountry == true {
		var country GeoIP2Country
		out.RegisteredCountry = &country
		out.RegisteredCountry.Name = in.RegisteredCountry.Names[language]
		out.RegisteredCountry.GeoNameId = in.RegisteredCountry.GeoNameID
		out.RegisteredCountry.Code = in.RegisteredCountry.IsoCode
	}
	if a.Factory.IncludeRepresentedCountry == true {
		var country GeoIP2Country
		out.RepresentedCountry = &country
		out.RepresentedCountry.Name = in.RepresentedCountry.Names[language]
		out.RepresentedCountry.GeoNameId = in.RepresentedCountry.GeoNameID
		out.RepresentedCountry.Code = in.RepresentedCountry.IsoCode
	}
	return &out
}


func (a *GeoIP2Annotator) GetFieldName() string {
	return "geoip2"
}

func (a *GeoIP2Annotator) Annotate(ip net.IP) interface{} {
	record, err := a.Reader.City(ip)
	if err != nil {
		log.Fatal(err)
	}
	return a.GeoIP2FillStruct(record)
}

func (a *GeoIP2Annotator) Close() error {
	return nil
}



func init() {
	f := new(GeoIP2AnnotatorFactory)
	RegisterAnnotator(f)
}
