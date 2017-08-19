package zannotate

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/osrg/gobgp/packet/mrt"
	"github.com/osrg/gobgp/table"
)

type mrtMessageCallback func(*mrt.MRTMessage)

func MrtRawIterate(filename string, cb mrtMessageCallback) error {
	file, err := os.Open(filename)
	fmt.Println("opening ", filename)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	for {
		buf := make([]byte, mrt.MRT_COMMON_HEADER_LEN)
		_, err := file.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("failed to read: %s", err)
		}
		h := &mrt.MRTHeader{}
		err = h.DecodeFromBytes(buf)
		if err != nil {
			log.Fatalf("failed to parse")
		}
		buf = make([]byte, h.Len)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatalf("failed to read")
		}
		msg, err := mrt.ParseMRTBody(h, buf)
		if err != nil {
			log.Fatalf("failed to parse: %s", err)
			continue
		}
		cb(msg)
	}
	return nil
}

// <dump-type>|<elem-type>|<record-ts>|<project>|<collector>|<peer-ASn>|<peer-IP>|<prefix>|<next-hop-IP>|<AS-path>|<origin-AS>|<communities>|<old-state>|<new-state>
// &{{1503007200 13 2 128} RIB: Seq [12509] Prefix [12.193.41.0/24] Entries [[RIB_ENTRY: PeerIndex [2] OriginatedTime [1502956719] PathAttrs [[{Origin: i} 7018 701 3385 10933 {Nexthop: 198.108.93.55} {LocalPref: 100} {Communities: 237:2, 237:7018}]] RIB_ENTRY: PeerIndex [3] OriginatedTime [1502956719] PathAttrs [[{Origin: i} 7018 701 3385 10933 {Nexthop: 198.108.93.58} {LocalPref: 100} {Communities: 237:2, 237:7018}]]]]}
type RIBOutput struct {
	Prefix  string
	NextHop string
	ASPath  []uint32
}

type mrtPathCallback func() string

func MrtPathIterate(filename string, cb mrtPathCallback) error {
	var peers []*mrt.Peer
	MrtRawIterate(filename, func(msg *mrt.MRTMessage) {
		if msg.Header.Type != mrt.TABLE_DUMPv2 {
			fmt.Println("something is fucked")
		}
		if msg.Header.Type == mrt.TABLE_DUMPv2 {
			subType := mrt.MRTSubTypeTableDumpv2(msg.Header.SubType)
			switch subType {
			case mrt.PEER_INDEX_TABLE:
				peers = msg.Body.(*mrt.PeerIndexTable).Peers
				fmt.Println("peer index table")
				fmt.Println(peers)
				return
			case mrt.RIB_IPV4_UNICAST, mrt.RIB_IPV6_UNICAST:
			case mrt.GEO_PEER_TABLE:
				fmt.Printf("WARNING: Skipping GEO_PEER_TABLE: %s", msg.Body.(*mrt.GeoPeerTable))
			default:
				log.Fatalf("unsupported subType: %v", subType)
			}

			if peers == nil {
				log.Fatalf("not found PEER_INDEX_TABLE")
			}

			rib := msg.Body.(*mrt.Rib)
			nlri := rib.Prefix

			paths := make([]*table.Path, 0, len(rib.Entries))

			for _, e := range rib.Entries {
				if len(peers) < int(e.PeerIndex) {
					log.Fatalf("invalid peer index: %d (PEER_INDEX_TABLE has only %d peers)\n",
						e.PeerIndex, len(peers))
				}
				source := &table.PeerInfo{
					AS: peers[e.PeerIndex].AS,
					ID: peers[e.PeerIndex].BgpId,
				}
				t := time.Unix(int64(e.OriginatedTime), 0)
				paths = append(paths, table.NewPath(source, nlri,
					false, e.PathAttributes, t, false))
			}
		}
	})
	return nil
}
