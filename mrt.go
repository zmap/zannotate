package zannotate

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/osrg/gobgp/packet/bgp"
	"github.com/osrg/gobgp/packet/mrt"
)

type mrtMessageCallback func(*mrt.MRTMessage)

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

func MrtRawIterate(filename string, cb mrtMessageCallback) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	for {
		buf := make([]byte, mrt.MRT_COMMON_HEADER_LEN)
		if _, err := file.Read(buf); err == io.EOF {
			return nil
		} else if err != nil {
			log.Fatalf("failed to read: %s", err)
		}
		h := &mrt.MRTHeader{}
		if err := h.DecodeFromBytes(buf); err != nil {
			log.Fatalf("failed to parse")
		}
		buf = make([]byte, h.Len)
		if _, err := file.Read(buf); err != nil {
			log.Fatalf("failed to read")
		}
		if msg, err := mrt.ParseMRTBody(h, buf); err != nil {
			log.Fatalf("failed to parse: %s", err)
		} else {
			cb(msg)
		}
	}
	return nil
}

// <dump-type>|<elem-type>|<record-ts>|<project>|<collector>|<peer-ASn>|<peer-IP>|<prefix>|<next-hop-IP>|<AS-path>|<origin-AS>|<communities>|<old-state>|<new-state>
// &{{1503007200 13 2 128} RIB: Seq [12509] Prefix [12.193.41.0/24] Entries [[RIB_ENTRY: PeerIndex [2] OriginatedTime [1502956719] PathAttrs [[{Origin: i} 7018 701 3385 10933 {Nexthop: 198.108.93.55} {LocalPref: 100} {Communities: 237:2, 237:7018}]] RIB_ENTRY: PeerIndex [3] OriginatedTime [1502956719] PathAttrs [[{Origin: i} 7018 701 3385 10933 {Nexthop: 198.108.93.58} {LocalPref: 100} {Communities: 237:2, 237:7018}]]]]}
type mrtPathCallback func(*RIBEntry)

type From struct {
	AS      uint32 `json:"as"`
	ID      net.IP `json:"id"`
	Address net.IP `json:"address"`
}

type Attributes struct {
	Origin          string   `json:"origin"`
	LocalPref       uint32   `json:"local_pref"`
	Communities     []string `json:"communities"`
	NextHop         net.IP   `json:"next_hop,omitempty"`
	OriginatorId    net.IP   `json:"originator_id,omitempty"`
	ASPath          []uint32 `json:"as_path"`
	AtomicAggregate bool     `json:"atomic_aggregate"`
	Aggregate       From     `json:"aggregate"`
	MultiExitDesc   uint32   `json:"multi_exit_desc,omitempty"`
}

type RIBEntry struct {
	Type           string          `json:"type"`
	SubType        string          `json:"sub_type"`
	SequenceNumber uint32          `json:"sequence_number"`
	Prefix         string          `json:"prefix"`
	From           From            `json:"from"`
	RouteFamily    bgp.RouteFamily `json:"route_family"`
	PeerIndex      uint16          `json:"peer_index"`
	OriginatedTime time.Time       `json:"orginated_time"`
	Timestamp      time.Time       `json:"timestamp"`
	PathIdentifier uint32          `json:"path_identifier"`
	Attributes     Attributes      `json:"attributes"`
}

func MrtPathIterate(filename string, cb mrtPathCallback) error {
	var peers []*mrt.Peer
	MrtRawIterate(filename, func(msg *mrt.MRTMessage) {
		if msg.Header.Type != mrt.TABLE_DUMPv2 {
			log.Fatal("MRT file is not a TABLE_DUMPv2")
		}
		subType := mrt.MRTSubTypeTableDumpv2(msg.Header.SubType)
		if subType == mrt.PEER_INDEX_TABLE {
			peers = msg.Body.(*mrt.PeerIndexTable).Peers
			return
		}
		if subType == mrt.GEO_PEER_TABLE {
			return
		}
		// we should have seen a peers table at this point
		// we need it to output any RIB entries
		if peers == nil {
			log.Fatalf("not found PEER_INDEX_TABLE")
		}
		rib := msg.Body.(*mrt.Rib)
		//nlri := rib.Prefix
		for _, e := range rib.Entries {
			if len(peers) < int(e.PeerIndex) {
				log.Fatalf("invalid peer index: %d (PEER_INDEX_TABLE has only %d peers)\n",
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

			out.OriginatedTime = time.Unix(int64(e.OriginatedTime), 0)
			out.Timestamp = time.Unix(int64(msg.Header.Timestamp), 0)
			// process attributes for additional data
			for _, a := range e.PathAttributes {
				if _, ok := a.(*bgp.PathAttributeAsPath); ok {

					//fmt.Println(as.Value)
					//params := make([]uint32, 0, len(as.Value))
					//for _, param := range as.Value {
					//	params = append(params, param)
					//}
					//out.Attributes.ASPath = params
				} else if nh, ok := a.(*bgp.PathAttributeNextHop); ok {
					out.Attributes.NextHop = nh.Value
				} else if meh, ok := a.(*bgp.PathAttributeMultiExitDisc); ok {
					out.Attributes.MultiExitDesc = meh.Value
				} else if lp, ok := a.(*bgp.PathAttributeLocalPref); ok {
					out.Attributes.LocalPref = lp.Value
				} else if agg, ok := a.(*bgp.PathAttributeAggregator); ok {
					// TODO
					fmt.Println(agg)
				} else if comm, ok := a.(*bgp.PathAttributeCommunities); ok {
					// TODO
					fmt.Println(comm)
				} else if orgId, ok := a.(*bgp.PathAttributeOriginatorId); ok {
					out.Attributes.OriginatorId = orgId.Value
				} else if mprnlri, ok := a.(*bgp.PathAttributeOrigin); ok {
					var typ string
					switch mprnlri.Value[0] {
					case bgp.BGP_ORIGIN_ATTR_TYPE_IGP:
						typ = "igp"
					case bgp.BGP_ORIGIN_ATTR_TYPE_EGP:
						typ = "egp"
					case bgp.BGP_ORIGIN_ATTR_TYPE_INCOMPLETE:
						typ = "incomplete"
					}
					out.Attributes.Origin = typ
					//} else if cl, ok := a.(*bgp.PathAttributeClusterList); ok {
					//	fmt.Println(cl)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeAtomicAggregate); ok {
					//	fmt.Println(mprnlri)
					//} else if mprnlri, ok := a.(*bgp.PathAttributeMpUnreachNLRI); ok {
					//	fmt.Println(mprnlri)
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
					//} else {
					//	log.Fatal("unknown", a.GetType())
					//}
					//} else if palp, ok := a.(*bgp.NewPathAttributeMpUnreachNLRI); ok {
				}
			}

			cb(&out)
		}
	})
	return nil
}
