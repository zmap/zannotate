package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/osrg/gobgp/packet/bgp"
	"github.com/osrg/gobgp/packet/mrt"
	log "github.com/sirupsen/logrus"
	"github.com/zmap/zannotate"
)

type MRT2JsonGlobalConf struct {
	InputFilePath  string
	OutputFilePath string
	LogFilePath    string
	Verbosity      int
}

type RawPeer struct {
	Type      string `json:"type"`
	BgpId     string `json:"bgp_id"`
	IpAddress string `json:"ip_address"`
	AS        uint32 `json:"as"`
}

type RawPeerIndexTableJSON struct {
	CollectorBgpId string     `json:"collector_bgp_id"`
	ViewName       string     `json:"view_name"`
	Peers          []*RawPeer `json:"peers"`
}

type RibEntry struct {
	PeerIndex      uint16                       `json:"peer_index"`
	OriginatedTime uint32                       `json:"orginated_time"`
	PathIdentifier uint32                       `json:"path_identifier"`
	PathAttributes []bgp.PathAttributeInterface `json:"path_attributes"`
	isAddPath      bool                         `json:"is_add_path"`
}

type Rib struct {
	SequenceNumber uint32                  `json:"sequence_number"`
	Prefix         bgp.AddrPrefixInterface `json:"prefix"`
	Entries        []*RibEntry             `json:"entries"`
	RouteFamily    bgp.RouteFamily         `json:"route_family"`
	isAddPath      bool                    `json:"is_add_path"`
}

func raw(conf *MRT2JsonGlobalConf) {
	f, err := os.OpenFile(conf.OutputFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Unable to open output file (%s): %s", conf.OutputFilePath, err.Error())
	}
	defer f.Close()
	zannotate.MrtRawIterate(conf.InputFilePath, func(msg *mrt.MRTMessage) {
		if msg.Header.Type != mrt.TABLE_DUMPv2 {
			fmt.Println("something is fucked")
		}
		if msg.Header.Type == mrt.TABLE_DUMPv2 {
			subType := mrt.MRTSubTypeTableDumpv2(msg.Header.SubType)
			switch subType {
			case mrt.PEER_INDEX_TABLE:
				var out RawPeerIndexTableJSON
				peers := msg.Body.(*mrt.PeerIndexTable).Peers
			case mrt.RIB_IPV4_UNICAST, mrt.RIB_IPV6_UNICAST:
				rib := msg.Body.(*mrt.Rib)
				fmt.Println(rib)
				//nlri := rib.Prefix

				//paths := make([]*table.Path, 0, len(rib.Entries))

				//for _, e := range rib.Entries {
				//	if len(peers) < int(e.PeerIndex) {
				//		log.Fatalf("invalid peer index: %d (PEER_INDEX_TABLE has only %d peers)\n",
				//			e.PeerIndex, len(peers))
				//	}
				//	source := &table.PeerInfo{
				//		AS: peers[e.PeerIndex].AS,
				//		ID: peers[e.PeerIndex].BgpId,
				//	}
				//	t := time.Unix(int64(e.OriginatedTime), 0)
				//	paths = append(paths, table.NewPath(source, nlri,
				//		false, e.PathAttributes, t, false))
				//}

			case mrt.GEO_PEER_TABLE:
				fmt.Printf("WARNING: Skipping GEO_PEER_TABLE: %s", msg.Body.(*mrt.GeoPeerTable))
			default:
				log.Fatalf("unsupported subType: %v", subType)
			}
		}

	})
}

func paths(conf *MRT2JsonGlobalConf) {
	zannotate.MrtRawIterate(conf.InputFilePath, func(msg *mrt.MRTMessage) {
		fmt.Println(msg)
	})
}

func main() {

	var conf MRT2JsonGlobalConf
	flags := flag.NewFlagSet("flags", flag.ExitOnError)
	flags.StringVar(&conf.InputFilePath, "input-file", "", "ip addresses to read")
	flags.StringVar(&conf.OutputFilePath, "output-file", "-", "where should JSON output be saved")
	flags.StringVar(&conf.LogFilePath, "log-file", "", "where should JSON output be saved")
	flags.IntVar(&conf.Verbosity, "verbosity", 3, "where should JSON output be saved")
	flags.Parse(os.Args[2:])

	fmt.Println(conf.InputFilePath)
	if conf.LogFilePath != "" {
		f, err := os.OpenFile(conf.LogFilePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Unable to open log file (%s): %s", conf.LogFilePath, err.Error())
		}
		log.SetOutput(f)
	}
	if conf.InputFilePath == "" {
		log.Fatal("no input path provided")
	}
	// Translate the assigned verbosity level to a logrus log level.
	switch conf.Verbosity {
	case 1: // Fatal
		log.SetLevel(log.FatalLevel)
	case 2: // Error
		log.SetLevel(log.ErrorLevel)
	case 3: // Warnings  (default)
		log.SetLevel(log.WarnLevel)
	case 4: // Information
		log.SetLevel(log.InfoLevel)
	case 5: // Debugging
		log.SetLevel(log.DebugLevel)
	default:
		log.Fatal("Unknown verbosity level specified. Must be between 1 (lowest)--5 (highest)")
	}
	if os.Args[1] == "raw" {
		raw(&conf)
	} else if os.Args[1] == "paths" {
		paths(&conf)
	} else {
		log.Fatal("invalid command")
	}

}
