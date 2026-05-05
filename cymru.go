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
	"errors"
	"flag"
	"fmt"
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	log "github.com/sirupsen/logrus"
	"github.com/zmap/dns"
	"github.com/zmap/zdns/v2/src/zdns"
)

type dnsTXTLookupFunc func(ctx context.Context, host string) ([]string, error)

// ASNLookup contains the result of a query to ASX.asn.cymru.com
type ASNLookup struct {
	ASN               uint32 `json:"asn,omitempty"`
	CountryCode       string `json:"country_code,omitempty"`
	Registry          string `json:"registry,omitempty"`
	ASNAllocationDate string `json:"asn_allocation_date,omitempty"`
	ASNDescription    string `json:"asn_description,omitempty"`
}

// CymruResult stores the format for the result from the Cymru annotator
type CymruResult struct {
	OriginASNs    []uint32                 `json:"origin_asns,omitempty"`
	PeerASNs      []uint32                 `json:"peer_asns,omitempty"`
	ASNLookup     map[uint32]*ASNLookup    `json:"asn_details,omitempty"`    // both Peer and Origin ASN Details
	PrefixDetails map[string]*PrefixResult `json:"prefix_details,omitempty"` // Prefix to details
}

type PrefixResult struct {
	Prefix         string   `json:"prefix,omitempty"`
	OriginASNs     []uint32 `json:"origin_asns,omitempty"`
	PeerASNs       []uint32 `json:"peer_asns,omitempty"`
	CountryCode    string   `json:"country_code,omitempty"`
	Registry       string   `json:"registry,omitempty"`
	AllocationDate string   `json:"allocation_date,omitempty"`
}

func (result *CymruResult) populateASNDetails(ctx context.Context, lookupFunc dnsTXTLookupFunc, originASN uint32) error {
	const asnURL = "asn.cymru.com"
	url := "AS" + strconv.Itoa(int(originASN)) + "." + asnURL
	resp, err := lookupFunc(ctx, url)
	if err != nil {
		return fmt.Errorf("could not lookup ASN %d: %w", originASN, err)
	}
	if len(resp) == 0 {
		return errors.New("no results found")
	}
	parts := strings.Split(resp[0], "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if len(parts) != 5 {
		return fmt.Errorf("asn endpoint returned unexpected result: %s", parts)
	}
	if result.ASNLookup == nil {
		result.ASNLookup = make(map[uint32]*ASNLookup)
	}
	result.ASNLookup[originASN] = &ASNLookup{
		ASN:               originASN,
		CountryCode:       parts[1],
		Registry:          parts[2],
		ASNAllocationDate: parts[3],
		ASNDescription:    parts[4],
	}
	return nil
}

func (result *CymruResult) populatePeerDetails(ctx context.Context, lookupFunc dnsTXTLookupFunc, ip net.IP) error {
	const peerURL = "peer.asn.cymru.com"
	url := convertIPToDNSFormat(ip) + "." + peerURL
	resp, err := lookupFunc(ctx, url)
	if err != nil {
		return err
	}
	if len(resp) == 0 {
		return errors.New("no results found")
	}
	peerASNs := make(map[uint32]struct{})
	for _, line := range resp {
		// Cymru can return multiple lines, each line corresponding to a prefix with potentially different peers
		parts := strings.Split(line, "|")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		if len(parts) != 5 {
			return fmt.Errorf("peer endpoint returned unexpected result: %s", parts)
		}
		prefix := strings.TrimSpace(parts[1])
		// PopulateOrigin will take care of most of the fields here, we're just interested in the peer ASNs
		prefixPeerASNs := make(map[uint32]struct{})
		for _, peer := range strings.Split(parts[0], " ") {
			if len(peer) == 0 {
				continue
			}
			peer = strings.TrimSpace(peer)
			var cast uint64
			cast, err = strconv.ParseUint(peer, 10, 32)
			if err != nil {
				return fmt.Errorf("peer endpoint returned peers (%s) that are invalid: %v", parts, err)
			}
			prefixPeerASNs[uint32(cast)] = struct{}{}
		}
		if len(result.PrefixDetails) == 0 {
			result.PrefixDetails = make(map[string]*PrefixResult)
		}
		details, ok := result.PrefixDetails[prefix]
		if !ok {
			details = &PrefixResult{
				Prefix:         prefix,
				CountryCode:    parts[2],
				Registry:       parts[3],
				AllocationDate: parts[4],
			}
		}
		details.PeerASNs = slices.Collect(maps.Keys(prefixPeerASNs))
		result.PrefixDetails[prefix] = details
		maps.Copy(peerASNs, prefixPeerASNs)
	}
	result.PeerASNs = slices.Collect(maps.Keys(peerASNs))
	return nil
}

func (result *CymruResult) populateOriginDetails(ctx context.Context, lookupFunc dnsTXTLookupFunc, ip net.IP) error {
	cymruOriginURL := "origin.asn.cymru.com"
	if ip.To4() == nil {
		// IPv4 uses a different URL
		cymruOriginURL = "origin6.asn.cymru.com"
	}
	url := convertIPToDNSFormat(ip) + "." + cymruOriginURL
	resp, err := lookupFunc(ctx, url)
	if err != nil {
		return fmt.Errorf("lookup failed for %s: %w", url, err)
	}
	if len(resp) == 0 {
		return fmt.Errorf("lookup returned no results for %s", url)
	}
	if len(result.PrefixDetails) == 0 {
		result.PrefixDetails = make(map[string]*PrefixResult)
	}
	originASNs := make(map[uint32]struct{})
	for _, line := range resp {
		prefixOriginASNs := make(map[uint32]struct{})
		parts := strings.Split(line, "|")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		if len(parts) != 5 {
			return fmt.Errorf("origin endpoint returned unexpected result: %s", parts)
		}
		prefixDetail, ok := result.PrefixDetails[parts[1]]
		if !ok {
			prefixDetail = &PrefixResult{
				Prefix:         parts[1],
				CountryCode:    parts[2],
				Registry:       parts[3],
				AllocationDate: parts[4],
			}
		}
		asnsStr := strings.Split(parts[0], " ")
		for _, originASN := range asnsStr {
			asn := strings.TrimSpace(originASN)
			asnInt, err := strconv.ParseUint(asn, 10, 32)
			if err != nil {
				return fmt.Errorf("origin ASN (%s) could not be parsed: %v", originASN, err)
			}
			prefixOriginASNs[uint32(asnInt)] = struct{}{}
		}
		prefixDetail.OriginASNs = slices.Collect(maps.Keys(prefixOriginASNs))
		maps.Copy(originASNs, prefixOriginASNs)
		result.PrefixDetails[parts[1]] = prefixDetail
	}
	result.OriginASNs = slices.Collect(maps.Keys(originASNs))
	return nil
}

type CymruAnnotatorFactory struct {
	BasePluginConf
	RawResolvers string
	zdnsConfig   *zdns.ResolverConfig
	timeoutSecs  int
	// We use our own cache here because ZDNS doens't cache TXT records internally
	cache           *lru.Cache[string, []string] // cache for TXT lookups, keyed by query URL
	queryOriginASN  bool
	queryPeerASN    bool
	queryASNDetails bool
}

type CymruAnnotator struct {
	Factory      *CymruAnnotatorFactory
	Id           int
	zdnsResolver *zdns.Resolver
	lookupFunc   dnsTXTLookupFunc
}

// Cymru Annotator Factory (Global)

func (a *CymruAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v CymruAnnotator
	v.Factory = a
	v.Id = i
	return &v
}

func (a *CymruAnnotatorFactory) Initialize(_ *GlobalConf) error {
	if noUserSpecifiedEnrichmentFlags := !a.queryOriginASN && !a.queryPeerASN && !a.queryASNDetails; noUserSpecifiedEnrichmentFlags {
		a.queryOriginASN = true
		a.queryPeerASN = true
		a.queryASNDetails = true
	}
	if userOnlySpecifiedASNDetails := a.queryASNDetails && !a.queryOriginASN && !a.queryPeerASN; userOnlySpecifiedASNDetails {
		return errors.New("cannot use --cymru-annotate-as-details without either --cymru-annotate-origin-as --cymru-annotate-peer-as as there'd be no ASs' to lookup")
	}
	a.zdnsConfig = zdns.NewResolverConfig()
	// Setup a common cache for all resolvers to use
	var err error
	a.cache, err = lru.New[string, []string](10000)
	if err != nil {
		return fmt.Errorf("could not create lru cache: %w", err)
	}
	if len(strings.TrimSpace(a.RawResolvers)) > 0 {
		for _, resolver := range strings.Split(a.RawResolvers, ",") {
			trimmed := strings.TrimSpace(resolver)
			ip := net.ParseIP(trimmed)
			if ip == nil {
				return fmt.Errorf("failed to parse dns server IP address: %s", trimmed)
			}
			ns := zdns.NameServer{IP: ip, Port: 53}
			if ip.To4() != nil {
				a.zdnsConfig.ExternalNameServersV4 = append(a.zdnsConfig.ExternalNameServersV4, ns)
			} else {
				a.zdnsConfig.ExternalNameServersV6 = append(a.zdnsConfig.ExternalNameServersV6, ns)
			}
		}
	}
	return nil
}

func (a *CymruAnnotatorFactory) GetWorkers() int {
	return a.Threads
}

func (a *CymruAnnotatorFactory) GroupName() string { return "Cymru" }

func (a *CymruAnnotatorFactory) Close() error {
	return nil
}

func (a *CymruAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *CymruAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "cymru", false, "enrich with Cymru's ASN and IP prefix data. Adds origin, peer, and ASN details by default, use the --cymru-annotate-... flags to specify a subset")
	flags.IntVar(&a.Threads, "cymru-threads", 50, "how many threads to use for Cymru lookups")
	flags.IntVar(&a.timeoutSecs, "cymru-timeout", 10, "timeout for each Cymru annotation, in seconds")
	flags.StringVar(&a.RawResolvers, "cymru-dns-servers", "", "list of DNS servers to use for Cymru TXT lookups, comma-separated IPs. If empty, uses system defaults")
	flags.BoolVar(&a.queryOriginASN, "cymru-annotate-origin-as", false, "enrich with Cymru's Origin AS data")
	flags.BoolVar(&a.queryPeerASN, "cymru-annotate-peer-as", false, "enrich with Cymru's Peer AS data, the possible BGP peers of the origin AS")
	flags.BoolVar(&a.queryASNDetails, "cymru-annotate-as-details", false, "enrich with ASN details for each peer/origin AS")
}

// Cymru Annotator (Per-Worker)
func (a *CymruAnnotator) Initialize() (err error) {
	if a.lookupFunc != nil {
		// mock lookup func being used
		return nil
	}
	a.zdnsResolver, err = zdns.InitResolver(a.Factory.zdnsConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize zdns resolver: %w", err)
	}
	a.lookupFunc = a.zdnsTXTLookup
	return nil
}

func (a *CymruAnnotator) zdnsTXTLookup(ctx context.Context, host string) ([]string, error) {
	// First, check cache
	// We use a cache here since the internal cache in zdns doesn't cache TXT records
	txts, ok := a.Factory.cache.Get(host)
	if ok {
		// Cache hit
		return txts, nil
	}
	// Cache Miss
	q := zdns.Question{
		Type:  dns.TypeTXT,
		Class: dns.ClassINET,
		Name:  host,
	}
	res, _, status, err := a.zdnsResolver.ExternalLookup(ctx, &q, nil)

	if status == zdns.StatusNXDomain {
		return nil, &net.DNSError{IsNotFound: true, Name: host}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lookup host %s with status %s: %w", host, status, err)
	}
	for _, answer := range res.Answers {
		if castAns, ok := answer.(zdns.Answer); ok && castAns.Type == "TXT" {
			txts = append(txts, castAns.Answer)
		}
	}
	if len(txts) > 0 {
		// Add to cache
		a.Factory.cache.Add(host, txts)
	}
	return txts, nil
}

func (a *CymruAnnotator) GetFieldName() string {
	return "cymru"
}

// Annotate performs a Cymru data lookup for the given IP address and returns the results.
// If an error occurs or a lookup fails, it returns nil
func (a *CymruAnnotator) Annotate(ip net.IP) interface{} {
	log.Debugf("IP (%s)in URL form: %s", ip.String(), convertIPToDNSFormat(ip))
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(a.Factory.timeoutSecs)*time.Second)
	defer cancelFn()
	result := &CymruResult{}
	var dnsErr *net.DNSError
	if a.Factory.queryOriginASN {
		err := result.populateOriginDetails(ctx, a.lookupFunc, ip)
		if err != nil && errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			// No record of this IP in Cymru, cannot continue
			log.Debugf("IP (%s) not found in cymru data", ip.String())
			return nil
		} else if err != nil {
			log.Debugf("error fetching cymru origin details for ip %s: %v", ip.String(), err)
			return nil
		}
	}
	if a.Factory.queryPeerASN {
		err := result.populatePeerDetails(ctx, a.lookupFunc, ip)
		if err != nil && errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			// No record of this IP in Cymru, cannot continue
			log.Debugf("IP (%s) not found in Cymru data", ip.String())
			return nil
		} else if err != nil {
			log.Debugf("error fetching cymru peer details for ip %s: %v", ip.String(), err)
			return nil
		}
	}
	if len(result.PeerASNs) == 0 && len(result.OriginASNs) == 0 {
		log.Debugf("no asns (peer and/or origin) found for ip %s in cymru origin lookup", ip.String())
		return nil
	}
	if a.Factory.queryASNDetails {
		hasASNLookedUp := make(map[uint32]struct{})
		for _, asn := range append(result.OriginASNs, result.PeerASNs...) {
			if _, ok := hasASNLookedUp[asn]; ok {
				continue // already seen before
			}
			hasASNLookedUp[asn] = struct{}{}
			err := result.populateASNDetails(ctx, a.lookupFunc, asn)
			if err != nil && errors.As(err, &dnsErr) && dnsErr.IsNotFound {
				// No record of this IP in Cymru, cannot continue
				log.Debugf("IP (%s) not found in Cymru data", ip.String())
				return nil
			} else if err != nil {
				log.Debugf("error fetching cymru ASN details for ip %s: %v", ip.String(), err)
			}
		}
	}
	return result
}

func (a *CymruAnnotator) Close() error {
	if a.zdnsResolver != nil {
		a.zdnsResolver.Close()
	}
	return nil
}

func init() {
	s := new(CymruAnnotatorFactory)
	RegisterAnnotator(s)
}

// convertIPToDNSFormat converts an IP into the string representation Cymru uses
// For IPv4, it wants the octets reversed with "." seperating
// For IPv6, queries are formed by reversing the nibbles of the address, and placing dots between each
// nibble, just like an IPv6 reverse DNS lookup"
func convertIPToDNSFormat(ip net.IP) string {
	if ipv4 := ip.To4(); ipv4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d", ipv4[3], ipv4[2], ipv4[1], ipv4[0])
	}
	ipLength := 16
	nibbles := make([]string, 0, ipLength*2) // IPv4: 4 bytes, IPv6: 16 bytes, each becomes 2 nibbles
	for i := ipLength - 1; i >= 0; i-- {
		// Extract low and high nibbles
		nibbles = append(nibbles, strconv.FormatUint(uint64(ip[i]&0x0f), 16)) // low nibble
		nibbles = append(nibbles, strconv.FormatUint(uint64(ip[i]>>4), 16))   // low nibble
	}
	return strings.Join(nibbles, ".")
}
