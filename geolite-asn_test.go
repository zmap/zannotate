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

func TestGeoIPASNAnnotator(t *testing.T) {
	tests := []struct {
		testName       string
		ipAddr         net.IP
		expectedResult *GeoLiteASNOutput
	}{
		{
			testName: "Positive Test Case, IPv4",
			ipAddr:   net.ParseIP("1.1.1.1"),
			expectedResult: &GeoLiteASNOutput{
				ASN:    13335,
				ASNOrg: "CLOUDFLARENET",
			},
		}, {
			testName: "Positive Test Case, IPv6",
			ipAddr:   net.ParseIP("2606:4700:4700::1111"),
			expectedResult: &GeoLiteASNOutput{
				ASN:    13335,
				ASNOrg: "CLOUDFLARENET",
			},
		}, {
			testName:       "Negative Test Case, Invalid IP",
			ipAddr:         net.ParseIP("999.999.999.999"),
			expectedResult: &GeoLiteASNOutput{},
		}, {
			testName:       "Negative Test Case, Private IP",
			ipAddr:         net.ParseIP("127.0.0.1"),
			expectedResult: &GeoLiteASNOutput{},
		},
	}
	factory := &GeoLiteASNAnnotatorFactory{
		Path: "./data-snapshots/geolite2_asn.mmdb",
	}
	err := factory.Initialize(nil)
	if err != nil {
		t.Fatalf("Failed to initialize factory: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			annotator := factory.MakeAnnotator(0).(*GeoLiteASNAnnotator)
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
