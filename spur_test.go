/*
 * ZAnnotate Copyright 2026 Regents of the University of Michigan
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
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type mockRoundTripper struct {
	expectedToken string
	status        int
	body          string
}

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Validate header was set (helps ensure Annotate sets Token)
	if req.Header.Get("Token") != m.expectedToken {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":"missing token"}`)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: m.status,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}, nil
}

func TestSpurAnnotatorMockSuccess(t *testing.T) {
	// Build factory and annotator
	factory := &SpurAnnotatorFactory{apiKey: "test-key", timeoutSecs: 2}
	a := factory.MakeAnnotator(0).(*SpurAnnotator)

	// Inject a mock http.Client that returns a fixed successful JSON body
	a.client = &http.Client{
		Transport: mockRoundTripper{
			expectedToken: "test-key",
			status:        http.StatusOK,
			body: `{"as":{"number":13335,"organization":"Cloudflare, Inc."},"infrastructure":"DATACENTER","ip":"1.1.1.1","location":{"city":"Anycast","country":"ZZ","state":"Anycast"},"organization":"Taguchi Digital Marketing System"}`,
		},
	}

	res := a.Annotate(net.ParseIP("1.1.1.1"))
	if res == nil {
		t.Fatalf("expected non-nil result")
	}

	raw, ok := res.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T", res)
	}

	expected := `{"as":{"number":13335,"organization":"Cloudflare, Inc."},"infrastructure":"DATACENTER","ip":"1.1.1.1","location":{"city":"Anycast","country":"ZZ","state":"Anycast"},"organization":"Taguchi Digital Marketing System"}`
	if string(raw) != expected {
		t.Fatalf("unexpected JSON returned: got %s want %s", string(raw), expected)
	}
}

func TestSpurAnnotatorMockNon200(t *testing.T) {
	factory := &SpurAnnotatorFactory{apiKey: "test-key", timeoutSecs: 2}
	a := factory.MakeAnnotator(0).(*SpurAnnotator)

	a.client = &http.Client{
		Transport: mockRoundTripper{
			expectedToken: "test-key",
			status:        http.StatusInternalServerError,
			body:          `{"error":"server error"}`,
		},
	}

	res := a.Annotate(net.ParseIP("1.1.1.1"))
	if res != nil {
		t.Fatalf("expected nil result for non-200 response, got %v", res)
	}
	// Test should return nil for a non-200 status
}
