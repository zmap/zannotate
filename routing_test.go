/* ZAnnotate Copyright 2026 Regents of the University of Michigan
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
	"bytes"
	"compress/bzip2"
	"io"
	"net"
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/zmap/zannotate/zrouting"
)

const MrtTestFile = "./data-snapshots/mrt_dump.bz2"

var tmpMrtFilePath string

func TestMain(m *testing.M) {
	// Read and decompress the bz2 file. We do this to save space in the git repo
	compressed, err := os.ReadFile(MrtTestFile)
	if err != nil {
		log.Fatalf("failed to read compressed test data: %v", err)
	}
	decompressed, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		log.Fatalf("failed to decompress test data: %v", err)
	}

	// Write to a temp file (works on Ubuntu, Mac, and GH runners)
	tmpFile, err := os.CreateTemp("", "mrt-*.bin")
	if err != nil {
		log.Fatalf("failed to create temp file: %v", err)
	}
	tmpMrtFilePath = tmpFile.Name()

	if _, err := tmpFile.Write(decompressed); err != nil {
		log.Fatalf("failed to write temp file: %v", err)
	}
	if err = tmpFile.Close(); err != nil {
		log.Fatalf("failed to close temp file: %v", err)
	}

	// Run tests, then clean up
	code := m.Run()
	err = os.Remove(tmpMrtFilePath)
	if err != nil {
		log.Errorf("failed to remove temp file (%s): %v", tmpMrtFilePath, err)
	}
	os.Exit(code)
}

func TestRoutingAnnotate(t *testing.T) {
	tests := []struct {
		testName         string // name of the subtest
		ip               net.IP // input IP address
		expectedResponse *zrouting.RoutingOutput
	}{
		{
			testName: "IPv4 Test Case",
			ip:       net.ParseIP("1.1.1.1"),
			expectedResponse: &zrouting.RoutingOutput{
				Prefix: "1.1.1.0/24",
				ASN:    13335,
				Path:   []uint32{4826, 13335},
				Origin: nil,
				Data:   nil,
			},
		}, {
			testName: "IPv6 Test Case",
			ip:       net.ParseIP("2001:4860:4860::8888"),
			expectedResponse: &zrouting.RoutingOutput{
				Prefix: "2001:4860::/32",
				ASN:    15169,
				Path:   []uint32{7594, 15169},
				Origin: nil,
				Data:   nil,
			},
		},
	}
	var factory RoutingAnnotatorFactory
	factory.RoutingTablePath = tmpMrtFilePath
	err := factory.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize Routing annotator factory: %v", err)
	}
	annotator := factory.MakeAnnotator(0)
	err = annotator.Initialize()
	if err != nil {
		t.Fatalf("failed to initialize Routing annotator: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			rawResult := annotator.Annotate(tt.ip)
			if rawResult == nil && tt.expectedResponse != nil {
				t.Errorf("Routing annotator returned a nil result, expected %v", tt.expectedResponse)
			}
			if !reflect.DeepEqual(rawResult, tt.expectedResponse) {
				t.Errorf("expected result %v, got %v", tt.expectedResponse, rawResult)
			}
		})
	}
}
