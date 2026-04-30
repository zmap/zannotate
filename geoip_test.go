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
	"net"
	"reflect"
	"testing"
)

func TestGeoIPAnnotator(t *testing.T) {
	tests := []struct {
		testName       string
		ipAddr         net.IP
		expectedResult *GeoIP2Output
	}{
		{
			testName: "Positive Test Case, IPv4",
			ipAddr:   net.ParseIP("1.2.3.4"),
			expectedResult: &GeoIP2Output{
				City: &GeoIP2City{},
				Country: &GeoIP2Country{
					Name:      "Australia",
					Code:      "AU",
					GeoNameId: 2077456,
				},
				Continent: &GeoIP2Country{
					Name:      "Oceania",
					Code:      "OC",
					GeoNameId: 6255151,
				},
				Postal:             &GeoIP2Postal{},
				LatLong:            &GeoIP2LatLong{},
				RepresentedCountry: &GeoIP2Country{},
				RegisteredCountry: &GeoIP2Country{
					Name:      "Australia",
					Code:      "AU",
					GeoNameId: 2077456,
				},
				Traits: &GeoIP2Traits{},
			},
		}, {
			testName: "Positive Test Case, IPv6",
			ipAddr:   net.ParseIP("2606:4700:4700::1111"),
			expectedResult: &GeoIP2Output{
				City:               &GeoIP2City{},
				Country:            &GeoIP2Country{},
				Continent:          &GeoIP2Country{},
				Postal:             &GeoIP2Postal{},
				LatLong:            &GeoIP2LatLong{},
				RepresentedCountry: &GeoIP2Country{},
				RegisteredCountry: &GeoIP2Country{
					Name:      "United States",
					Code:      "US",
					GeoNameId: 6252001,
				},
				Traits: &GeoIP2Traits{},
			},
		}, {
			testName:       "Negative Test Case, Private IP",
			ipAddr:         net.ParseIP("192.168.0.1"),
			expectedResult: &GeoIP2Output{
				City:               &GeoIP2City{ },
				Country:            &GeoIP2Country{ },
				Continent:          &GeoIP2Country{},
				Postal:             &GeoIP2Postal{},
				LatLong:            &GeoIP2LatLong{},
				RepresentedCountry: &GeoIP2Country{},
				RegisteredCountry:  &GeoIP2Country{},
				Traits:             &GeoIP2Traits{},
			},
		},
	}
	factory := &GeoIP2AnnotatorFactory{
		Path: "./data-snapshots/geolite2_country.mmdb",
		Mode: "mmap",
		Language: "en",
		RawInclude: "*",
	}
	err := factory.Initialize(nil)
	if err != nil {
		t.Fatalf("Failed to initialize factory: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			annotator := factory.MakeAnnotator(0).(*GeoIP2Annotator)
			err = annotator.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize annotator: %v", err)
			}
			result := annotator.Annotate(tt.ipAddr)
			if tt.expectedResult == nil && result == nil {
				return // pass
			}
			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("Annotating IP %s gave = %v; expected: %v", tt.ipAddr, result, tt.expectedResult)
			}
		})
	}
}
