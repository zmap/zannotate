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
	"math/rand/v2"
	"net"
	"reflect"
	"testing"
)

// TestGreyNoiseAnnotator tests that given a database file and a known IP, the annotator returns the expected values for that IP in the DB
func TestGreyNoiseAnnotator(t *testing.T) {
	expectedFields := map[string]any{
		"actor":              "Stanford University",
		"classification":     "benign",
		"date":               "2026-04-07",
		"handshake_complete": true,
		"last_seen":          "2026-04-07T00:00:00Z",
		"seen":               true,
		"tags":               []any{"Stanford University", "RDP Crawler", "RDP Protocol"},
	}
	factory := &GreyNoiseAnnotatorFactory{DBPath: "./data-snapshots/greynoise.mmdb"}
	err := factory.Initialize(nil)
	if err != nil {
		t.Fatalf("Error initializing greynoise annotator factory: %v", err)
	}
	a := factory.MakeAnnotator(0).(*GreyNoiseAnnotator)
	err = a.Initialize()
	if err != nil {
		t.Fatalf("Error initializing greynoise annotator: %v", err)
	}

	ip := "171.67.71.209"
	res := a.Annotate(net.ParseIP(ip))
	if res == nil {
		t.Fatalf("GreyNoiseAnnotator failed to annotate %s", ip)
	}

	for field, expected := range expectedFields {
		actual, ok := res.(map[string]any)[field]
		if !ok {
			t.Errorf("missing expected field %q", field)
			continue
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("field %q: expected %v (%T), got %v (%T)", field, expected, expected, actual, actual)
		}
	}
}

func BenchmarkGreyNoiseAnnotator(b *testing.B) {
	factory := &GreyNoiseAnnotatorFactory{DBPath: "./data-snapshots/greynoise.mmdb"}
	err := factory.Initialize(nil)
	if err != nil {
		b.Fatalf("Error initializing greynoise annotator factory: %v", err)
	}
	a := factory.MakeAnnotator(0).(*GreyNoiseAnnotator)
	err = a.Initialize()
	if err != nil {
		b.Fatalf("Error initializing greynoise annotator: %v", err)
	}

	// Pre-generate random IPs so generation is not part of the benchmark
	ips := make([]net.IP, 1000)
	for i := range ips {
		ips[i] = net.IPv4(
			byte(rand.IntN(256)),
			byte(rand.IntN(256)),
			byte(rand.IntN(256)),
			byte(rand.IntN(256)),
		)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		a.Annotate(ips[i%len(ips)])
	}
}
