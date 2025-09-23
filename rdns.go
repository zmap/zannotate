/*
 * ZAnnotate Copyright 2018 Regents of the University of Michigan
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
	"flag"
	"net"
)

type RDNSAnnotatorFactory struct {
	BasePluginConf
	RawResolvers string
}

type RDNSAnnotator struct {
	Factory *RDNSAnnotatorFactory
	Id      int
}

// RDNS Annotator Factory (Global)

func (a *RDNSAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v RDNSAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *RDNSAnnotatorFactory) Initialize(conf *GlobalConf) error {
	return nil
}

func (a *RDNSAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *RDNSAnnotatorFactory) Close() error {
	return nil
}

func (a *RDNSAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *RDNSAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	// Reverse DNS Lookup
	flags.BoolVar(&a.Enabled, "rdns", false, "reverse dns lookup")
	flags.StringVar(&a.RawResolvers, "rdns-dns-servers", "", "list of DNS servers to use for DNS lookups")
	flags.IntVar(&a.Threads, "rdns-threads", 100, "how many reverse dns threads")
}

// RDNS Annotator (Per-Worker)

func (a *RDNSAnnotator) Initialize() error {
	return nil
}

func (a *RDNSAnnotator) GetFieldName() string {
	return "rdns"
}

func (a *RDNSAnnotator) Annotate(ip net.IP) interface{} {
	return nil
}

func (a *RDNSAnnotator) Close() error {
	return nil
}

func init() {
	s := new(RDNSAnnotatorFactory)
	RegisterAnnotator(s)
}
