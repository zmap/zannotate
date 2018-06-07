/*
 * ZAnnotate Copyright 2017 Regents of the University of Michigan
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

package zrouting

import (
	"encoding/json"
	"io"
	"net"

	"github.com/osrg/gobgp/packet/bgp"
	log "github.com/sirupsen/logrus"
	"github.com/zmap/go-iptree/iptree"
	"github.com/zmap/zannotate/zmrt"
)

type ASNameNode struct {
	ASN          uint32 `json:"asn,omitempty"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	Organization string `json:"organization,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
}

type ASTreeNode struct {
	Prefix string
	ASN    uint32
	Path   []uint32
}

type RoutingOutput struct {
	Prefix string       `json:"prefix"`
	ASN    uint32       `json:"asn,omitempty"`
	Path   []uint32     `json:"path,omitempty"`
	Origin *ASNameNode  `json:"as,omitempty"`
	Data   *interface{} `json:"data,omitempty"`
}

type RoutingLookupTree struct {
	ASNames map[uint32]ASNameNode
	ASData  map[uint32]interface{}
	IPTree  *iptree.IPTree
}

//	RoutingTablePath string
//	ASNamesPath      string

func (t *RoutingLookupTree) PopulateFromMRT(raw io.Reader) {
	// build radix tree and populate with
	t.IPTree = iptree.New()
	zmrt.MrtPathIterate(raw, func(e *zmrt.RIBEntry) {
		if e.AFI == bgp.AFI_IP {
			var n ASTreeNode
			n.Prefix = e.Prefix
			n.Path = e.Attributes.ASPath
			if len(n.Path) > 0 {
				n.ASN = n.Path[len(n.Path)-1]
			}
			t.IPTree.AddByString(e.Prefix, n)
		}
	})
}

func (t *RoutingLookupTree) SetASName(asn uint32, m ASNameNode) {
	if t.ASNames == nil {
		t.ASNames = make(map[uint32]ASNameNode)
	}
	t.ASNames[asn] = m
}

func (t *RoutingLookupTree) SetASData(asn uint32, m interface{}) {
	if t.ASData == nil {
		t.ASData = make(map[uint32]interface{})
	}
	t.ASData[asn] = m
}

func (t *RoutingLookupTree) PopulateASnames(raw io.Reader) {
	d := json.NewDecoder(raw)
	for {
		var m ASNameNode
		if err := d.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("%s", err)
		}
		t.SetASName(m.ASN, m)
	}
}

func (t *RoutingLookupTree) Get(ip net.IP) (*RoutingOutput, error) {
	var out RoutingOutput
	if n, ok, err := t.IPTree.Get(ip); ok && err == nil {
		node := n.(ASTreeNode)
		out.Prefix = node.Prefix
		out.Path = node.Path
		out.ASN = node.ASN
		if t.ASNames != nil {
			var n ASNameNode
			if name, ok := t.ASNames[out.ASN]; ok {
				n.Description = name.Description
				n.Organization = name.Organization
				n.Name = name.Name
				n.CountryCode = name.CountryCode
				n.ASN = node.ASN
				out.Origin = &n
			}
		}
		return &out, nil
	} else {
		return nil, err
	}
}
