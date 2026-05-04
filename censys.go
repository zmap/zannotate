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
	"flag"
	"io"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type CensysAnnotatorFactory struct {
	BasePluginConf
	client        *http.Client // Shared client across threads
	personalToken string       // User's personal access token for API auth
}

// Censys Annotator Factory (Global)

func (a *CensysAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v CensysAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *CensysAnnotatorFactory) Initialize(_ *GlobalConf) error {
	a.client = http.DefaultClient
	return nil
}

func (a *CensysAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *CensysAnnotatorFactory) Close() error {
	return nil
}

func (a *CensysAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *CensysAnnotatorFactory) GroupName() string { return "Censys" }

func (a *CensysAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "censys", false, "censys internet intelligence")
	flags.StringVar(&a.personalToken, "censys-pat", "", "censys API personal access token (PAT)")
	flags.IntVar(&a.Threads, "censys-threads", 1, "how many enrichment threads to use. Note that free plan only allows 1 concurrent API request at a time")
}

// CensysAnnotator (Per-Worker)
type CensysAnnotator struct {
	Factory *CensysAnnotatorFactory
	Id      int
}

func (a *CensysAnnotator) Initialize() (err error) {
	return nil
}

func (a *CensysAnnotator) GetFieldName() string {
	return "censys"
}

var censysAPIHostLookupURL = "https://api.platform.censys.io/v3/global/asset/host/"

// Annotate performs a Censys host lookup for the given IP address and returns the results.
// If an error occurs or a lookup fails, it returns nil
func (a *CensysAnnotator) Annotate(ip net.IP) interface{} {

	req, err := http.NewRequest("GET", censysAPIHostLookupURL+ip.String(), nil)
	if err != nil {
		// If we can't even form a request, we'll fail to enrich anything. Erroring out.
		log.Fatalf("could not form an http request for enriching with censys data for ip %s: %v", ip.String(), err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+a.Factory.personalToken)

	res, err := a.Factory.client.Do(req)
	if err != nil {
		log.Debugf("failed to annotate ip %s with censys: %v", ip.String(), err)
		return nil
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Debugf("failed to close response body: %v", err)
		}
	}(res.Body)
	if res.StatusCode != http.StatusOK {
		log.Debugf("failed to annotate ip %s with censys: %s", ip.String(), res.Status)
		return nil
	}
	body, _ := io.ReadAll(res.Body)
	var result any
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Debugf("failed to parse censys response for ip %s: %v", ip.String(), err)
		return nil
	}
	// By default, Censys' result is wrapped in a result[resource[real_data]]. We'll unwrap that here
	dropResultCast, ok := result.(map[string]interface{})
	if !ok {
		log.Debugf("failed to unwrap censys response for ip %s: %v", ip.String(), result)
		return result
	}
	dropResult := dropResultCast["result"]
	if dropResult == nil {
		log.Debugf("failed to unwrap censys response for ip %s: %v", ip.String(), result)
		return result
	}
	dropResourceCast, ok := dropResult.(map[string]interface{})
	if !ok {
		log.Debugf("failed to unwrap censys response for ip %s: %v", ip.String(), result)
		return result
	}
	dropResource := dropResourceCast["resource"]
	if dropResource == nil {
		log.Debugf("failed to unwrap censys response for ip %s: %v", ip.String(), result)
		return result
	}
	return dropResource
}

func (a *CensysAnnotator) Close() error {
	return nil
}

func init() {
	s := new(CensysAnnotatorFactory)
	RegisterAnnotator(s)
}
