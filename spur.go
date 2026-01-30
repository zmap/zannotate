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
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const SPUR_API_URL = "https://api.spur.us/v2/context/"

type SpurAnnotatorFactory struct {
	BasePluginConf
	apiKey      string // Spur API Key, pulled from env var
	timeoutSecs int
}

type SpurAnnotator struct {
	Factory *SpurAnnotatorFactory
	Id      int
	client  *http.Client
}

// Spur Annotator Factory (Global)

func (a *SpurAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v SpurAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *SpurAnnotatorFactory) Initialize(_ *GlobalConf) error {
	// Check for API key
	a.apiKey = os.Getenv("SPUR_API_KEY")
	if len(a.apiKey) == 0 {
		return errors.New("SPUR_API_KEY environment variable not set. Please use 'export SPUR_API_KEY=your_api_key' to set it")
	}
	return nil
}

func (a *SpurAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *SpurAnnotatorFactory) Close() error {
	return nil
}

func (a *SpurAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *SpurAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "spur", false, "enrich with Spur's threat intelligence data")
	flags.IntVar(&a.Threads, "spur-threads", 100, "how many threads to use for Spur lookups")
	flags.IntVar(&a.timeoutSecs, "spur-timeout", 2, "timeout for each Spur query, in seconds")
}

// Spur Annotator (Per-Worker)
func (a *SpurAnnotator) Initialize() error {
	a.client = &http.Client{
		Timeout: time.Duration(a.Factory.timeoutSecs) * time.Second,
	}
	return nil
}

func (a *SpurAnnotator) GetFieldName() string {
	return "spur"
}

// Annotate performs a Spur data lookup for the given IP address and returns the results.
// If an error occurs or a lookup fails, it returns nil
func (a *SpurAnnotator) Annotate(ip net.IP) interface{} {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s%s", SPUR_API_URL, ip.String()),
		nil,
	)
	if err != nil {
		log.Errorf("failed to create Spur HTTP request for IP %s: %v", ip.String(), err)
		return nil
	}

	req.Header.Set("Token", a.Factory.apiKey) // Set the API key in the request header

	resp, err := a.client.Do(req)
	if err != nil {
		log.Errorf("http request to Spur API failed for IP %s: %v", ip.String(), err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response body for IP %s: %v", ip.String(), err)
		return nil
	}
	if resp.StatusCode == http.StatusOK {
		trimmed, _ := strings.CutSuffix(string(body), "\n") // Remove trailing newline if present, cleans up output
		return json.RawMessage(trimmed)
	}

	log.Errorf("Spur API returned non-200 status for IP %s: %d - %s", ip.String(), resp.StatusCode, string(body))

	return nil
}

func (a *SpurAnnotator) Close() error {
	return nil
}

func init() {
	s := new(SpurAnnotatorFactory)
	RegisterAnnotator(s)
}
