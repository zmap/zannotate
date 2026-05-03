/*
 * ZAnnotate Copyright 2026 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the License); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an AS IS BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package zannotate

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"
)

func TestConvertIPToDNSFormat(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected string
	}{
		{
			name:     "ipv4 simple",
			ip:       net.ParseIP("1.1.1.1"),
			expected: "1.1.1.1",
		},
		{
			name:     "ipv4 asymmetric",
			ip:       net.ParseIP("171.67.71.209").To4(),
			expected: "209.71.67.171",
		},
		{
			name:     "ipv6 cloudflare",
			ip:       net.ParseIP("2001:4860:b002::68").To16(),
			expected: "8.6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.b.0.6.8.4.1.0.0.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertIPToDNSFormat(tt.ip)
			if result != tt.expected {
				t.Errorf("convertIPToDNSFormat(%v) = %q, want %q", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestPopulateOriginDetails(t *testing.T) {
	tests := []struct {
		name        string
		ip          net.IP
		mockResult  []string
		mockErr     error
		expectError bool
		validate    func(*testing.T, *CymruResult)
	}{
		{
			name:       "valid origin result",
			ip:         net.ParseIP("171.64.1.1"),
			mockResult: []string{"32 | 171.64.0.0/14 | US | arin | 1994-08-22"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if result.OriginASNs != "32" {
					t.Errorf("expected OriginASN 32, got %s", result.OriginASNs)
				}
				if result.PrefixDetails.Prefix != "171.64.0.0/14" {
					t.Errorf("expected Prefix 171.64.0.0/14, got %s", result.PrefixDetails.Prefix)
				}
				if result.PrefixDetails.CountryCode != "US" {
					t.Errorf("expected CountryCode US, got %s", result.PrefixDetails.CountryCode)
				}
				if result.PrefixDetails.Registry != "arin" {
					t.Errorf("expected Registry arin, got %s", result.PrefixDetails.Registry)
				}
				if result.PrefixDetails.AllocationDate != "1994-08-22" {
					t.Errorf("expected AllocationDate 1994-08-22, got %s", result.PrefixDetails.AllocationDate)
				}
			},
		},
		{
			name:       "valid origin result, 2+ p",
			ip:         net.ParseIP("34.114.10.22"),
			mockResult: []string{"15169 19527 | 34.112.0.0/14 | US | arin | 2018-09-28"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				expectedOriginASNs := []string{"15169", "19527"}
				if reflect.DeepEqual(result.OriginASNs, expectedOriginASNs) {
					t.Errorf("expected OriginASN %v, got %s", expectedOriginASNs, result.OriginASNs)
				}
				if result.PrefixDetails.Prefix != "34.112.0.0/14" {
					t.Errorf("expected Prefix 34.112.0.0/14, got %s", result.PrefixDetails.Prefix)
				}
				if result.PrefixDetails.CountryCode != "US" {
					t.Errorf("expected CountryCode US, got %s", result.PrefixDetails.CountryCode)
				}
				if result.PrefixDetails.Registry != "arin" {
					t.Errorf("expected Registry arin, got %s", result.PrefixDetails.Registry)
				}
				if result.PrefixDetails.AllocationDate != "2018-09-28" {
					t.Errorf("expected AllocationDate 1994-08-22, got %s", result.PrefixDetails.AllocationDate)
				}
			},
		},
		{
			name:        "no results from DNS",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too few fields",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"32 | 171.64.0.0/14 | US"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too many fields",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"32 | 171.64.0.0/14 | US | arin | 1994-08-22 | extra"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "invalid ASN (non-numeric)",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"invalid | 171.64.0.0/14 | US | arin | 1994-08-22"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:       "ASN with extra whitespace",
			ip:         net.ParseIP("171.64.1.1"),
			mockResult: []string{"  32  |  171.64.0.0/14  |  US  |  arin  |  1994-08-22  "},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if result.OriginASNs != "32" {
					t.Errorf("expected OriginASN 32 after trimming, got %s", result.OriginASNs)
				}
				if result.PrefixDetails.CountryCode != "US" {
					t.Errorf("expected CountryCode US after trimming, got %s", result.PrefixDetails.CountryCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CymruResult{}
			lookupFunc := func(ctx context.Context, name string) ([]string, error) {
				return tt.mockResult, tt.mockErr
			}
			err := result.populateOriginDetails(context.Background(), lookupFunc, tt.ip)
			if (err != nil) != tt.expectError {
				t.Errorf("populateOriginDetails() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestPopulatePeerDetails(t *testing.T) {
	tests := []struct {
		name        string
		ip          net.IP
		mockResult  []string
		mockErr     error
		expectError bool
		validate    func(*testing.T, *CymruResult)
	}{
		{
			name:       "valid peer result with single ASN",
			ip:         net.ParseIP("171.64.1.1"),
			mockResult: []string{"46749 | 171.64.0.0/14 | US | arin | 1994-08-22"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if len(result.PeerASNs) != 1 {
					t.Errorf("expected 1 peer ASN, got %d", len(result.PeerASNs))
					return
				}
				if result.PeerASNs[0] != 46749 {
					t.Errorf("expected PeerASN 46749, got %d", result.PeerASNs[0])
				}
			},
		},
		{
			name:       "valid peer result with multiple ASNs",
			ip:         net.ParseIP("1.1.1.1"),
			mockResult: []string{"2914 6461 6939 13335 23352 | 1.1.1.0/24 | AU | apnic | 2011-08-11"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if len(result.PeerASNs) != 5 {
					t.Errorf("expected 5 peer ASNs, got %d", len(result.PeerASNs))
					return
				}
				expected := []uint32{2914, 6461, 6939, 13335, 23352}
				for i, exp := range expected {
					if result.PeerASNs[i] != exp {
						t.Errorf("peer ASN[%d]: expected %d, got %d", i, exp, result.PeerASNs[i])
					}
				}
			},
		},
		{
			name:        "no results from DNS",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too few fields",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"2914 6461 | 1.1.1.0/24 | AU"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "invalid peer ASN (non-numeric)",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"2914 invalid 6939 | 1.1.1.0/24 | AU | apnic | 2011-08-11"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:       "peer ASNs with extra whitespace",
			ip:         net.ParseIP("1.1.1.1"),
			mockResult: []string{"  2914   6461   6939  |  1.1.1.0/24  |  AU  |  apnic  |  2011-08-11  "},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if len(result.PeerASNs) != 3 {
					t.Errorf("expected 3 peer ASNs after trimming, got %d", len(result.PeerASNs))
					return
				}
				expected := []uint32{2914, 6461, 6939}
				for i, exp := range expected {
					if result.PeerASNs[i] != exp {
						t.Errorf("peer ASN[%d] after trimming: expected %d, got %d", i, exp, result.PeerASNs[i])
					}
				}
			},
		},
		{
			name:        "ASN value exceeds uint32",
			ip:          net.ParseIP("10.0.0.1"),
			mockResult:  []string{"9999999999999 | 1.1.1.0/24 | AU | apnic | 2011-08-11"},
			mockErr:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CymruResult{}
			lookupFunc := func(ctx context.Context, name string) ([]string, error) {
				return tt.mockResult, tt.mockErr
			}
			err := result.populatePeerDetails(context.Background(), lookupFunc, tt.ip)
			if (err != nil) != tt.expectError {
				t.Errorf("populatePeerDetails() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestPopulateASNDetails(t *testing.T) {
	tests := []struct {
		name        string
		originASN   string
		mockResult  []string
		mockErr     error
		expectError bool
		validate    func(*testing.T, *CymruResult)
	}{
		{
			name:       "valid ASN result",
			originASN:  "32",
			mockResult: []string{"32 | US | arin | 1984-09-24 | STANFORD - Stanford University, US"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if result.ASNLookup.ASN != 32 {
					t.Errorf("expected ASN 32, got %d", result.ASNLookup.ASN)
				}
				if result.ASNLookup.CountryCode != "US" {
					t.Errorf("expected CountryCode US, got %s", result.ASNLookup.CountryCode)
				}
				if result.ASNLookup.Registry != "arin" {
					t.Errorf("expected Registry arin, got %s", result.ASNLookup.Registry)
				}
				if result.ASNLookup.ASNAllocationDate != "1984-09-24" {
					t.Errorf("expected ASNAllocationDate 1984-09-24, got %s", result.ASNLookup.ASNAllocationDate)
				}
				if result.ASNLookup.ASNDescription != "STANFORD - Stanford University, US" {
					t.Errorf("expected ASNDescription, got %s", result.ASNLookup.ASNDescription)
				}
			},
		},
		{
			name:       "ASN with long description",
			originASN:  "13335",
			mockResult: []string{"13335 | US | arin | 2010-07-14 | CLOUDFLARENET - Cloudflare, Inc., US"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if result.ASNLookup.ASN != 13335 {
					t.Errorf("expected ASN 13335, got %d", result.ASNLookup.ASN)
				}
				if result.ASNLookup.ASNDescription != "CLOUDFLARENET - Cloudflare, Inc., US" {
					t.Errorf("expected full description, got %s", result.ASNLookup.ASNDescription)
				}
			},
		},
		{
			name:        "no results from DNS",
			originASN:   "32",
			mockResult:  []string{},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too few fields",
			originASN:   "32",
			mockResult:  []string{"32 | US | arin"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too many fields",
			originASN:   "32",
			mockResult:  []string{"32 | US | arin | 1984-09-24 | STANFORD - Stanford University, US | extra"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "invalid ASN (non-numeric)",
			originASN:   "invalid",
			mockResult:  []string{"invalid | US | arin | 1984-09-24 | STANFORD - Stanford University, US"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:       "ASN with extra whitespace",
			originASN:  "32",
			mockResult: []string{"  32  |  US  |  arin  |  1984-09-24  |  STANFORD - Stanford University, US  "},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				if result.ASNLookup.ASN != 32 {
					t.Errorf("expected ASN 32 after trimming, got %d", result.ASNLookup.ASN)
				}
				if result.ASNLookup.CountryCode != "US" {
					t.Errorf("expected CountryCode US after trimming, got %s", result.ASNLookup.CountryCode)
				}
				if result.ASNLookup.ASNDescription != "STANFORD - Stanford University, US" {
					t.Errorf("expected trimmed description, got %s", result.ASNLookup.ASNDescription)
				}
			},
		},
		{
			name:        "ASN value exceeds uint32",
			originASN:   "9999999999999",
			mockResult:  []string{"9999999999999 | US | arin | 1984-09-24 | STANFORD - Stanford University, US"},
			mockErr:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CymruResult{}
			lookupFunc := func(ctx context.Context, name string) ([]string, error) {
				return tt.mockResult, tt.mockErr
			}

			err := result.populateASNDetails(context.Background(), lookupFunc, tt.originASN)
			if (err != nil) != tt.expectError {
				t.Errorf("populateASNDetails() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCymruAnnotate(t *testing.T) {
	type lookupResults struct {
		originResp string
		peerResp   string
		asnResp    string
	}
	tests := []struct {
		testName             string // name of the subtest
		ip                   net.IP // input IP address
		hardCodedLookupResps lookupResults
		expectedResponse     *CymruResult
	}{
		{
			testName: "Positive IPv4 Test Case",
			ip:       net.ParseIP("1.1.1.1"),
			hardCodedLookupResps: lookupResults{
				originResp: "13335 | 1.1.1.0/24 | AU | apnic | 2011-08-11",
				peerResp:   "2914 6461 6939 13335 23352 | 1.1.1.0/24 | AU | apnic | 2011-08-11",
				asnResp:    "13335 | US | arin | 2010-07-14 | CLOUDFLARENET - Cloudflare, Inc., US",
			},
			expectedResponse: &CymruResult{
				OriginASNs: "13335",
				PeerASNs:   []uint32{2914, 6461, 6939, 13335, 23352},
				ASNLookup: &ASNLookup{
					ASN:               13335,
					CountryCode:       "US",
					Registry:          "arin",
					ASNAllocationDate: "2010-07-14",
					ASNDescription:    "CLOUDFLARENET - Cloudflare, Inc., US",
				},
				PrefixDetails: &PrefixResult{
					Prefix:         "1.1.1.0/24",
					CountryCode:    "AU",
					Registry:       "apnic",
					AllocationDate: "2011-08-11",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			lookupFunc := func(ctx context.Context, name string) ([]string, error) {
				if strings.HasSuffix(name, "origin.asn.cymru.com") {
					return []string{test.hardCodedLookupResps.originResp}, nil
				} else if strings.HasSuffix(name, "peer.asn.cymru.com") {
					return []string{test.hardCodedLookupResps.peerResp}, nil
				} else if strings.HasSuffix(name, "asn.cymru.com") {
					if !strings.HasPrefix(name, "AS") {
						t.Fatalf("url is incorrect: %s", name)
					}
					return []string{test.hardCodedLookupResps.asnResp}, nil
				}
				t.Fatalf("url is incorrect: %s", name)
				return nil, nil
			}
			var factory CymruAnnotatorFactory
			factory.mockDNSFunc = lookupFunc
			err := factory.Initialize(nil)
			if err != nil {
				t.Fatalf("failed to initialize Cymru annotator factory: %v", err)
			}
			annotator := factory.MakeAnnotator(0)
			err = annotator.Initialize()
			if err != nil {
				t.Fatalf("failed to initialize Cymru annotator: %v", err)
			}
			rawResult := annotator.Annotate(test.ip)
			if !reflect.DeepEqual(rawResult, test.expectedResponse) {
				t.Fatalf("unexpected result: got %v want %v", rawResult, test.expectedResponse)
			}
		})
	}
}
