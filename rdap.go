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
	"context"
	"flag"
	"net"
	"time"

	"github.com/openrdap/rdap"
)

type RDAPAnnotatorFactory struct {
	BasePluginConf
	Timeout int // Timeout for each RDAP query, in seconds
}

type RDAPAnnotator struct {
	Factory    *RDAPAnnotatorFactory
	Id         int
	rdapClient *rdap.Client
}

// RDAP Annotator Factory (Global)
func (a *RDAPAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "rdap", false, "annotate with RDAP (successor to WHOIS) lookup")
	flags.IntVar(&a.Threads, "rdap-threads", 5, "how many rdap processing threads to use")
	flags.IntVar(&a.Timeout, "rdap-timeout", 2, "RDAP query timeout in seconds")
}

func (a *RDAPAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *RDAPAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *RDAPAnnotatorFactory) Initialize(_ *GlobalConf) error {
	return nil
}

func (a *RDAPAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v RDAPAnnotator
	v.Factory = a
	v.Id = i
	v.rdapClient = &rdap.Client{
		HTTP:                      nil,
		Bootstrap:                 nil,
		Verbose:                   nil,
		ServiceProviderExperiment: false,
		UserAgent:                 "",
	}
	return &v
}

func (a *RDAPAnnotatorFactory) Close() error {
	return nil
}

// Routing Annotator (Per-Worker)

func (a *RDAPAnnotator) Initialize() error {
	return nil
}

func (a *RDAPAnnotator) GetFieldName() string {
	return "rdap/whois"
}

func (a *RDAPAnnotator) Annotate(ip net.IP) interface{} {
	req := rdap.NewIPRequest(ip)
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(a.Factory.Timeout)*time.Second))
	defer cancelFunc()
	req = req.WithContext(ctx)
	resp, err := a.rdapClient.Do(req)
	if err != nil {
		return nil
	}
	if len(resp.HTTP) == 0 {
		return nil
	}

	return resp.Object
}

func (a *RDAPAnnotator) Close() error {
	return nil
}

func init() {
	s := new(RDAPAnnotatorFactory)
	RegisterAnnotator(s)
}
