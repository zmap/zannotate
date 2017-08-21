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

package zannotate

import (
	"net"

	"github.com/osrg/gobgp/packet/bgp"
	"github.com/zmap/go-iptree/iptree"
	"github.com/zmap/zannotate/zmrt"
)

type ASNameNode struct {
	ASN         uint32 `json:"asn"`
	Description string `json:"as_description"`
	Name        string `json:"as_name"`
}

type ASTreeNode struct {
	Prefix      string   `json:"prefix"`
	ASN         uint32   `json:"asn,omitempty"`
	Path        []uint32 `json:"path,omitempty"`
	Description string   `json:"as_description,omitempty"`
	Name        string   `json:"as_name,omitempty"`
}

type RoutingConf struct {
	RoutingTablePath string
	ASNamesPath      string
	ASNames          map[uint32]ASNameNode
	IPTree           *iptree.IPTree
}

type RoutingOutput struct {
	ASTreeNode
}

func BuildTree(conf *RoutingConf) {
	if conf.ASNamesPath != "" {

	}
	// build radix tree and populate with
	conf.IPTree = iptree.New()
	zmrt.MrtPathIterate(conf.RoutingTablePath, func(e *zmrt.RIBEntry) {
		if e.AFI == bgp.AFI_IP {
			var n ASTreeNode
			n.Prefix = e.Prefix
			n.Path = e.Attributes.ASPath
			if len(n.Path) > 0 {
				n.ASN = n.Path[len(n.Path)-1]
			}
			conf.IPTree.AddByString(e.Prefix, n)
		}
	})
}

func RoutingFillStruct(ip net.IP, conf *RoutingConf) *RoutingOutput {
	var out RoutingOutput
	if n, ok, _ := conf.IPTree.Get(ip); ok {
		node := n.(ASTreeNode)
		out.Prefix = node.Prefix
		out.Path = node.Path
		out.ASN = node.ASN
		out.Description = node.Description
		out.Name = node.Name
	}
	return &out
}
