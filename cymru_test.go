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
	"cmp"
	"context"
	"net"
	"reflect"
	"slices"
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
			name:     "ipv6",
			ip:       net.ParseIP("2001:abcd:ef12:b002::68").To16(),
			expected: "8.6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.2.0.0.b.2.1.f.e.d.c.b.a.1.0.0.2",
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
				expected := &PrefixResult{
					Prefix:         "171.64.0.0/14",
					CountryCode:    "US",
					Registry:       "arin",
					AllocationDate: "1994-08-22",
					OriginASNs:     []uint32{32},
				}
				if !reflect.DeepEqual(result.OriginASNs, expected.OriginASNs) {
					t.Errorf("expected OriginASNs %v, got %v", expected.OriginASNs, result.OriginASNs)
				}
				if !reflect.DeepEqual(result.PrefixDetails[0], expected) {
					t.Errorf("expected PrefixDetails[171.64.0.0/14] %v, got %v", expected, result.PrefixDetails[0])
				}
			},
		},
		{
			name:       "valid origin result, 2+ origin asns",
			ip:         net.ParseIP("34.114.10.22"),
			mockResult: []string{"15169 19527 | 34.112.0.0/14 | US | arin | 2018-09-28"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				slices.Sort(result.OriginASNs)
				pd := result.PrefixDetails[0]
				slices.Sort(pd.OriginASNs)
				expected := &PrefixResult{
					Prefix:         "34.112.0.0/14",
					CountryCode:    "US",
					Registry:       "arin",
					AllocationDate: "2018-09-28",
					OriginASNs:     []uint32{15169, 19527},
				}
				if !reflect.DeepEqual(result.OriginASNs, expected.OriginASNs) {
					t.Errorf("expected OriginASNs %v, got %v", expected.OriginASNs, result.OriginASNs)
				}
				if !reflect.DeepEqual(pd, expected) {
					t.Errorf("expected PrefixDetails[34.112.0.0/14] %v, got %v", expected, pd)
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
				expectedASNs := []uint32{32}
				if !reflect.DeepEqual(result.OriginASNs, expectedASNs) {
					t.Errorf("expected OriginASNs %v after trimming, got %v", expectedASNs, result.OriginASNs)
				}
				if result.PrefixDetails[0].CountryCode != "US" {
					t.Errorf("expected CountryCode US after trimming, got %s", result.PrefixDetails[0].CountryCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CymruResult{
				asnLookupMap:    make(map[uint32]struct{}),
				prefixLookupMap: make(map[string]*PrefixResult),
			}
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
				expected := &PrefixResult{
					Prefix:         "171.64.0.0/14",
					CountryCode:    "US",
					Registry:       "arin",
					AllocationDate: "1994-08-22",
					PeerASNs:       []uint32{46749},
				}
				if !reflect.DeepEqual(result.PeerASNs, expected.PeerASNs) {
					t.Errorf("expected PeerASNs %v, got %v", expected.PeerASNs, result.PeerASNs)
				}
				if !reflect.DeepEqual(result.PrefixDetails[0], expected) {
					t.Errorf("expected PrefixDetails[171.64.0.0/14] %v, got %v", expected, result.PrefixDetails[0])
				}
			},
		},
		{
			name:       "valid peer result with multiple ASNs",
			ip:         net.ParseIP("1.1.1.1"),
			mockResult: []string{"2914 6461 6939 13335 23352 | 1.1.1.0/24 | AU | apnic | 2011-08-11"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				slices.Sort(result.PeerASNs)
				pd := result.PrefixDetails[0]
				slices.Sort(pd.PeerASNs)
				expected := &PrefixResult{
					Prefix:         "1.1.1.0/24",
					CountryCode:    "AU",
					Registry:       "apnic",
					AllocationDate: "2011-08-11",
					PeerASNs:       []uint32{2914, 6461, 6939, 13335, 23352},
				}
				if !reflect.DeepEqual(result.PeerASNs, expected.PeerASNs) {
					t.Errorf("expected PeerASNs %v, got %v", expected.PeerASNs, result.PeerASNs)
				}
				if !reflect.DeepEqual(pd, expected) {
					t.Errorf("expected PrefixDetails[1.1.1.0/24] %v, got %v", expected, pd)
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
				slices.Sort(result.PeerASNs)
				expected := []uint32{2914, 6461, 6939}
				if !reflect.DeepEqual(result.PeerASNs, expected) {
					t.Errorf("expected PeerASNs %v after trimming, got %v", expected, result.PeerASNs)
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
			result := &CymruResult{
				asnLookupMap:    make(map[uint32]struct{}),
				prefixLookupMap: make(map[string]*PrefixResult),
			}
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
		originASN   uint32
		mockResult  []string
		mockErr     error
		expectError bool
		validate    func(*testing.T, *CymruResult)
	}{
		{
			name:       "valid ASN result",
			originASN:  32,
			mockResult: []string{"32 | US | arin | 1984-09-24 | STANFORD - Stanford University, US"},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				expected := []*ASNLookup{
					{ASN: 32, CountryCode: "US", Registry: "arin", ASNAllocationDate: "1984-09-24", ASNDescription: "STANFORD - Stanford University, US"},
				}
				if !reflect.DeepEqual(result.ASNLookup, expected) {
					t.Errorf("expected ASNLookup %v, got %v", expected, result.ASNLookup)
				}
			},
		},
		{
			name:        "no results from DNS",
			originASN:   32,
			mockResult:  []string{},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too few fields",
			originASN:   32,
			mockResult:  []string{"32 | US | arin"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:        "too many fields",
			originASN:   32,
			mockResult:  []string{"32 | US | arin | 1984-09-24 | STANFORD - Stanford University, US | extra"},
			mockErr:     nil,
			expectError: true,
		},
		{
			name:       "ASN with extra whitespace",
			originASN:  32,
			mockResult: []string{"  32  |  US  |  arin  |  1984-09-24  |  STANFORD - Stanford University, US  "},
			mockErr:    nil,
			validate: func(t *testing.T, result *CymruResult) {
				expected := []*ASNLookup{
					{ASN: 32, CountryCode: "US", Registry: "arin", ASNAllocationDate: "1984-09-24", ASNDescription: "STANFORD - Stanford University, US"},
				}
				if !reflect.DeepEqual(result.ASNLookup, expected) {
					t.Errorf("expected ASNLookup %v after trimming, got %v", expected, result.ASNLookup)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CymruResult{
				asnLookupMap:    make(map[uint32]struct{}),
				prefixLookupMap: make(map[string]*PrefixResult),
			}
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

func normalizeCymruResult(r *CymruResult) {
	if r == nil {
		return
	}
	slices.Sort(r.OriginASNs)
	slices.Sort(r.PeerASNs)
	slices.SortFunc(r.ASNLookup, func(i, j *ASNLookup) int {
		return cmp.Compare(i.ASN, j.ASN)
	})
	slices.SortFunc(r.PrefixDetails, func(i, j *PrefixResult) int {
		return strings.Compare(i.Prefix, j.Prefix)
	})
	for _, pd := range r.PrefixDetails {
		slices.Sort(pd.OriginASNs)
		slices.Sort(pd.PeerASNs)
	}
	r.prefixLookupMap = nil
	r.asnLookupMap = nil
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
				OriginASNs: []uint32{13335},
				PeerASNs:   []uint32{2914, 6461, 6939, 13335, 23352},
				// Annotate looks up ASN details for all origin + peer ASNs; the mock
				// returns the same Cloudflare record for every query, so each entry
				// carries that record's metadata but with the correct ASN key/field.
				ASNLookup: []*ASNLookup{
					{ASN: 13335, CountryCode: "US", Registry: "arin", ASNAllocationDate: "2010-07-14", ASNDescription: "CLOUDFLARENET - Cloudflare, Inc., US"},
					{ASN: 2914, CountryCode: "US", Registry: "arin", ASNAllocationDate: "2010-07-14", ASNDescription: "CLOUDFLARENET - Cloudflare, Inc., US"},
					{ASN: 6461, CountryCode: "US", Registry: "arin", ASNAllocationDate: "2010-07-14", ASNDescription: "CLOUDFLARENET - Cloudflare, Inc., US"},
					{ASN: 6939, CountryCode: "US", Registry: "arin", ASNAllocationDate: "2010-07-14", ASNDescription: "CLOUDFLARENET - Cloudflare, Inc., US"},
					{ASN: 23352, CountryCode: "US", Registry: "arin", ASNAllocationDate: "2010-07-14", ASNDescription: "CLOUDFLARENET - Cloudflare, Inc., US"},
				},
				PrefixDetails: []*PrefixResult{
					{
						Prefix:         "1.1.1.0/24",
						CountryCode:    "AU",
						Registry:       "apnic",
						AllocationDate: "2011-08-11",
						OriginASNs:     []uint32{13335},
						PeerASNs:       []uint32{2914, 6461, 6939, 13335, 23352},
					},
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
			err := factory.Initialize(nil)
			if err != nil {
				t.Fatalf("failed to initialize Cymru annotator factory: %v", err)
			}
			annotator := factory.MakeAnnotator(0)
			if cymruAnnotator, ok := annotator.(*CymruAnnotator); ok {
				cymruAnnotator.lookupFunc = lookupFunc
			}
			err = annotator.Initialize()
			if err != nil {
				t.Fatalf("failed to initialize Cymru annotator: %v", err)
			}
			actual := annotator.Annotate(test.ip).(*CymruResult)
			normalizeCymruResult(actual)
			normalizeCymruResult(test.expectedResponse)
			if !reflect.DeepEqual(actual, test.expectedResponse) {
				t.Fatalf("unexpected result: got %v want %v", actual, test.expectedResponse)
			}
		})
	}
}
