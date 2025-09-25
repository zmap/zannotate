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

func TestRDNSAnnotate(t *testing.T) {
	tests := []struct {
		testName    string // name of the subtest
		ip          net.IP // input IP address
		expectNull  bool   // whether we expect a null result, ie. on local network
		domainNames []string
	}{
		{
			testName:    "Positive Test Case",
			ip:          net.ParseIP("1.1.1.1"),
			expectNull:  false,
			domainNames: []string{"one.one.one.one"},
		}, {
			testName:    "Negative Test Case",
			ip:          net.ParseIP("127.0.0.1"),
			expectNull:  true,
			domainNames: nil,
		}, {
			testName:    "IPv6 Test Case",
			ip:          net.ParseIP("2001:4860:4860::8888"),
			expectNull:  false,
			domainNames: []string{"dns.google"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			var factory RDNSAnnotatorFactory

			err := factory.Initialize(nil)
			if err != nil {
				t.Fatalf("failed to initialize RDNS annotator factory: %v", err)
			}
			annotator := factory.MakeAnnotator(0)
			err = annotator.Initialize()
			if err != nil {
				t.Fatalf("failed to initialize RDNS annotator: %v", err)
			}
			rawResult := annotator.Annotate(tt.ip)
			if tt.expectNull != (rawResult == nil) {
				t.Errorf("expected null: %v, got: %v", tt.expectNull, rawResult)
			}
			if tt.expectNull {
				return // pass, expected null and got null
			}
			// check type and content
			result, ok := rawResult.(RDNSOutput)
			if !ok {
				t.Fatalf("expected RDNSOutput type, got: %T", rawResult)
			}
			if !reflect.DeepEqual(result.DomainNames, tt.domainNames) {
				t.Errorf("expected domain names %v, got %v", tt.domainNames, result.DomainNames)
			}
		})
	}
}
