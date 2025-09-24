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
	"context"
	"flag"
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/zmap/dns"
	"github.com/zmap/zdns/v2/src/zdns"
)

type RDNSOutput struct {
	DomainNames []string `json:"domain_names, omitempty"`
	Status      string   `json:"status,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type RDNSAnnotatorFactory struct {
	BasePluginConf
	RawResolvers string
	zdnsConfig   *zdns.ResolverConfig
}

type RDNSAnnotator struct {
	Factory      *RDNSAnnotatorFactory
	Id           int
	zdnsResolver *zdns.Resolver
}

// RDNS Annotator Factory (Global)

func (a *RDNSAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v RDNSAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *RDNSAnnotatorFactory) Initialize(conf *GlobalConf) error {
	a.zdnsConfig = zdns.NewResolverConfig()
	if len(strings.TrimSpace(a.RawResolvers)) > 0 {
		// Parse and Validate the User-Specified Resolvers
		// 1. split on comma
		resolvers := strings.Split(a.RawResolvers, ",")
		// 2. trim whitespace
		for _, resolver := range resolvers {
			trimmedString := strings.TrimSpace(resolver)
			// 3. validate IP
			ip := net.ParseIP(trimmedString)
			if ip == nil {
				return fmt.Errorf("failed to parse dns server IP address: %s", trimmedString)
			}
			// 4. Differentiate between IPv4 and IPv6
			ns := zdns.NameServer{
				IP:         ip,
				Port:       53,
				DomainName: "",
			}
			if ip.To4() != nil {
				a.zdnsConfig.ExternalNameServersV4 = append(a.zdnsConfig.ExternalNameServersV4, ns)
			} else {
				a.zdnsConfig.ExternalNameServersV6 = append(a.zdnsConfig.ExternalNameServersV6, ns)
			}
		}
	}

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
	flags.StringVar(&a.RawResolvers, "rdns-dns-servers", "", "list of DNS servers to use for DNS lookups, comma-separated IP list. If empty, will use system defaults")
	flags.IntVar(&a.Threads, "rdns-threads", 100, "how many reverse dns threads")
}

// RDNS Annotator (Per-Worker)

func (a *RDNSAnnotator) Initialize() (err error) {
	a.zdnsResolver, err = zdns.InitResolver(a.Factory.zdnsConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize zdns resolver: %w", err)
	}
	return nil
}

func (a *RDNSAnnotator) GetFieldName() string {
	return "rdns"
}

func (a *RDNSAnnotator) Annotate(ip net.IP) interface{} {
	q := zdns.Question{
		Type:  dns.TypePTR,
		Class: dns.ClassINET,
		Name:  ip.String(),
	}
	output := RDNSOutput{}
	res, _, status, err := a.zdnsResolver.ExternalLookup(context.Background(), &q, nil)
	// TODO Phillip - check if the other modules handle errors by including the error in output, or failing silently as a best-effort
	if err != nil {
		output.Error = err.Error()
		return output
	}
	if res == nil {
		log.Fatalf("zdns returned a nil result without erroring, zannotate cannot continue") // this should never happen, but this will be more helpful than a panic
	}
	output.Status = string(status)
	output.DomainNames = make([]string, 0, len(res.Answers))
	for _, answer := range res.Answers {
		if castAns, ok := answer.(zdns.Answer); ok {
			output.DomainNames = append(output.DomainNames, strings.TrimSuffix(castAns.Answer, ".")) // remove trailing period
		}
	}
	return output
}

func (a *RDNSAnnotator) Close() error {
	return nil
}

func init() {
	s := new(RDNSAnnotatorFactory)
	RegisterAnnotator(s)
}
