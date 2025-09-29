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

// TestIPInfoAnnotator tests the IPInfoAnnotator with known IP addresses against a constant MMDB file in ./data-snapshots
// Unfortunately, I only have a free account for IPInfo, so can only test parsing for the Lite api with a real DB file.
func TestIPInfoAnnotator(t *testing.T) {
	tests := []struct {
		testName       string
		ipAddr         net.IP
		expectedResult *IPInfoOutput
	}{
		{
			testName: "Positive Test Case, IPv4",
			ipAddr:   net.ParseIP("1.1.1.1"),
			expectedResult: &IPInfoOutput{
				ASN:           "AS13335",
				ASName:        "Cloudflare, Inc.",
				ASDomain:      "cloudflare.com",
				Country:       "Australia",
				CountryCode:   "AU",
				Continent:     "Oceania",
				ContinentCode: "OC",
			},
		}, {
			testName: "Positive Test Case, IPv6",
			ipAddr:   net.ParseIP("2606:4700:4700::1111"),
			expectedResult: &IPInfoOutput{
				ASN:           "AS13335",
				ASName:        "Cloudflare, Inc.",
				ASDomain:      "cloudflare.com",
				Country:       "United States",
				CountryCode:   "US",
				Continent:     "North America",
				ContinentCode: "NA",
			},
		}, {
			testName:       "Negative Test Case, Invalid IP",
			ipAddr:         net.ParseIP("999.999.999.999"),
			expectedResult: nil,
		}, {
			testName:       "Negative Test Case, Private IP",
			ipAddr:         net.ParseIP("127.0.0.1"),
			expectedResult: nil,
		},
	}
	factory := &IPInfoAnnotatorFactory{
		DatabaseFilePath: "./data-snapshots/ipinfo_lite.mmdb",
	}
	err := factory.Initialize(nil)
	if err != nil {
		t.Fatalf("Failed to initialize IPInfoAnnotatorFactory: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			annotator := factory.MakeAnnotator(0).(*IPInfoAnnotator)
			err = annotator.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize IPInfoAnnotator: %v", err)
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
