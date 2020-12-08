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

package zmrt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/packet/mrt"
	"github.com/sirupsen/logrus"
)

type mrtMessageCallback func(*mrt.MRTMessage) error

func MrtTypeToName(t mrt.MRTType) string {
	switch t {
	case mrt.NULL:
		return "null"
	case mrt.START:
		return "start"
	case mrt.DIE:
		return "die"
	case mrt.I_AM_DEAD:
		return "i_am_dead"
	case mrt.PEER_DOWN:
		return "peer_down"
	case mrt.BGP:
		return "bgp"
	case mrt.RIP:
		return "rip"
	case mrt.IDRP:
		return "idrp"
	case mrt.RIPNG:
		return "riping"
	case mrt.BGP4PLUS:
		return "bgp4plus"
	case mrt.BGP4PLUS01:
		return "bgp4plus01"
	case mrt.OSPFv2:
		return "ospf_v2"
	case mrt.TABLE_DUMP:
		return "table_dump"
	case mrt.TABLE_DUMPv2:
		return "table_dump_v2"
	case mrt.BGP4MP:
		return "bgp4mp"
	case mrt.BGP4MP_ET:
		return "bgp4mp_et"
	case mrt.ISIS:
		return "isis"
	case mrt.ISIS_ET:
		return "isis_et"
	case mrt.OSPFv3:
		return "ospf_v3"
	case mrt.OSPFv3_ET:
		return "ospf_v3_et"
	default:
		return "unknown"
	}
}

func MrtSubTypeToName(t uint16) string {
	switch mrt.MRTSubTypeTableDumpv2(t) {
	case mrt.RIB_IPV4_UNICAST:
		return "rib_ipv4_unicast"
	case mrt.RIB_IPV6_UNICAST:
		return "rib_ipv6_unicast"
	case mrt.RIB_IPV6_MULTICAST:
		return "rib_ipv6_multicast"
	case mrt.RIB_GENERIC:
		return "rib_generic"
	case mrt.RIB_IPV4_UNICAST_ADDPATH:
		return "rib_ipv4_unicast_addpath"
	case mrt.RIB_IPV4_MULTICAST_ADDPATH:
		return "rib_ipv4_multicast_addpath"
	case mrt.RIB_IPV6_UNICAST_ADDPATH:
		return "rib_ipv6_unicast_addpath"
	case mrt.RIB_IPV6_MULTICAST_ADDPATH:
		return "rib_ipv6_multicast_addpath"
	case mrt.RIB_GENERIC_ADDPATH:
		return "rib_generic_addpath"
	default:
		return "unknown"
	}
}

func MrtRawIterate(raw io.Reader, cb mrtMessageCallback) error {
	buffered := bufio.NewReader(raw)
	var n int
	var err error
	for {
		buf := make([]byte, mrt.MRT_COMMON_HEADER_LEN)
		if n, err = io.ReadFull(buffered, buf); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read: %s", err)
		}
		h := &mrt.MRTHeader{}
		if err := h.DecodeFromBytes(buf[0:n]); err != nil {
			return errors.New("failed to parse")
		}
		buf = make([]byte, h.Len)
		if n, err = io.ReadFull(buffered, buf); err != nil {
			return errors.New("failed to read")
		}
		if msg, err := mrt.ParseMRTBody(h, buf[0:n]); err != nil {
			return fmt.Errorf("failed to parse: %s", err)
		} else {
			if err := cb(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

type mrtPathCallback func(*RIBEntry)

type From struct {
	AS      uint32 `json:"as,omitempty"`
	ID      net.IP `json:"id,omitempty"`
	Address net.IP `json:"address,omitempty"`
}

type Attributes struct {
	Origin          string                          `json:"origin,omitempty"`
	LocalPref       uint32                          `json:"local_pref,omitempty"`
	Communities     []string                        `json:"communities,omitempty"`
	NextHop         net.IP                          `json:"next_hop,omitempty"`
	OriginatorId    net.IP                          `json:"originator_id,omitempty"`
	ASPath          []uint32                        `json:"as_path,omitempty"`
	AtomicAggregate bool                            `json:"atomic_aggregate"`
	Aggregator      *From                           `json:"aggregator,omitempty"`
	MultiExitDesc   uint32                          `json:"multi_exit_desc,omitempty"`
	MpReachNLRI     *bgp.PathAttributeMpReachNLRI   `json:"mp_reach_nlri,omitempty"`
	MpUnreachNLRI   *bgp.PathAttributeMpUnreachNLRI `json:"mp_unreach_nlri,omitempty"`
}

type RIBEntry struct {
	Type           string `json:"type"`
	SubType        string `json:"sub_type"`
	SequenceNumber uint32 `json:"sequence_number"`
	Prefix         string `json:"prefix"`
	From           From   `json:"peer"`
	AFI            uint16
	//RouteFamily    bgp.RouteFamily `json:"route_family"`
	PeerIndex      uint16     `json:"peer_index"`
	OriginatedTime time.Time  `json:"orginated_time"`
	Timestamp      time.Time  `json:"timestamp"`
	PathIdentifier uint32     `json:"path_identifier"`
	Attributes     Attributes `json:"attributes"`
}

func MrtPathIterate(raw io.Reader, cb mrtPathCallback) error {
	var peers []*mrt.Peer
	if err := MrtRawIterate(raw, func(msg *mrt.MRTMessage) error {
		if msg.Header.Type != mrt.TABLE_DUMPv2 {
			return errors.New("MRT file is not a TABLE_DUMPv2")
		}
		subType := mrt.MRTSubTypeTableDumpv2(msg.Header.SubType)
		if subType == mrt.PEER_INDEX_TABLE {
			peers = msg.Body.(*mrt.PeerIndexTable).Peers
			return nil
		}
		if subType == mrt.GEO_PEER_TABLE {
			return nil
		}
		// we should have seen a peers table at this point
		// we need it to output any RIB entries
		if peers == nil {
			return errors.New("not found PEER_INDEX_TABLE")
		}
		rib := msg.Body.(*mrt.Rib)
		//nlri := rib.Prefix
		for _, e := range rib.Entries {
			if len(peers) < int(e.PeerIndex) {
				return fmt.Errorf("invalid peer index: %d (PEER_INDEX_TABLE has only %d peers)\n",
					e.PeerIndex, len(peers))
			}
			// create reasonable output
			var out RIBEntry
			out.Type = MrtTypeToName(msg.Header.Type)
			out.SubType = MrtSubTypeToName(msg.Header.SubType)
			out.SequenceNumber = rib.SequenceNumber
			out.Prefix = rib.Prefix.String()
			out.From.AS = peers[e.PeerIndex].AS
			out.From.ID = peers[e.PeerIndex].BgpId
			out.From.Address = peers[e.PeerIndex].IpAddress

			out.PeerIndex = e.PeerIndex
			out.PathIdentifier = e.PathIdentifier
			out.AFI = rib.Prefix.AFI()

			out.OriginatedTime = time.Unix(int64(e.OriginatedTime), 0)
			out.Timestamp = time.Unix(int64(msg.Header.Timestamp), 0)
			// process attributes for additional data
			for _, a := range e.PathAttributes {
				if as, ok := a.(*bgp.PathAttributeAsPath); ok {
					for _, param := range as.Value {
						if p, ok := param.(*bgp.As4PathParam); ok {
							out.Attributes.ASPath = p.AS
						} else {
							return errors.New("unknown AS path type")
						}
					}
				} else if nh, ok := a.(*bgp.PathAttributeNextHop); ok {
					out.Attributes.NextHop = nh.Value
				} else if meh, ok := a.(*bgp.PathAttributeMultiExitDisc); ok {
					out.Attributes.MultiExitDesc = meh.Value
				} else if lp, ok := a.(*bgp.PathAttributeLocalPref); ok {
					out.Attributes.LocalPref = lp.Value
				} else if agg, ok := a.(*bgp.PathAttributeAggregator); ok {
					var f From
					f.AS = agg.Value.AS
					f.Address = agg.Value.Address
					out.Attributes.Aggregator = &f
				} else if comm, ok := a.(*bgp.PathAttributeCommunities); ok {
					l := []string{}
					for _, v := range comm.Value {
						n, ok := bgp.WellKnownCommunityNameMap[bgp.WellKnownCommunity(v)]
						if ok {
							l = append(l, n)
						} else {
							l = append(l, fmt.Sprintf("%d:%d", (0xffff0000&v)>>16, 0xffff&v))
						}
					}
					out.Attributes.Communities = l
				} else if orgId, ok := a.(*bgp.PathAttributeOriginatorId); ok {
					out.Attributes.OriginatorId = orgId.Value
				} else if origin, ok := a.(*bgp.PathAttributeOrigin); ok {
					v := uint8(origin.Value)
					var typ string
					switch v {
					case bgp.BGP_ORIGIN_ATTR_TYPE_IGP:
						typ = "igp"
					case bgp.BGP_ORIGIN_ATTR_TYPE_EGP:
						typ = "egp"
					case bgp.BGP_ORIGIN_ATTR_TYPE_INCOMPLETE:
						typ = "incomplete"
					}
					out.Attributes.Origin = typ
				} else if _, ok := a.(*bgp.PathAttributeAtomicAggregate); ok {
					out.Attributes.AtomicAggregate = true
				} else if mprnlri, ok := a.(*bgp.PathAttributeMpReachNLRI); ok {
					out.Attributes.MpReachNLRI = mprnlri
				} else if mprnlri, ok := a.(*bgp.PathAttributeMpUnreachNLRI); ok {
					out.Attributes.MpUnreachNLRI = mprnlri
					//} else if cl, ok := a.(*bgp.PathAttributeClusterList); ok {
					//	fmt.Println(cl)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeExtendedCommunities); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeAs4Path); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeAs4Aggregator); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeTunnelEncap); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributePmsiTunnel); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeIP6ExtendedCommunities); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeAigp); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeLargeCommunities); ok {
					//	fmt.Println(mprnlri)
					//} else if palp, ok := a.(*bgp.NewPathAttributeMpUnreachNLRI); ok {
				} else {
					logrus.Debugf("unsupported attribute type: %v", a.GetType())
				}
			}
			cb(&out)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
