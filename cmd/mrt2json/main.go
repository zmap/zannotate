package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/osrg/gobgp/packet/bgp"
	"github.com/osrg/gobgp/packet/mrt"
	log "github.com/sirupsen/logrus"
	"github.com/zmap/zannotate/zmrt"
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
	Type           string     `json:"type"`
}

type RawRibEntry struct {
	PeerIndex      uint16                       `json:"peer_index"`
	OriginatedTime uint32                       `json:"orginated_time"`
	PathIdentifier uint32                       `json:"path_identifier"`
	PathAttributes []bgp.PathAttributeInterface `json:"path_attributes"`
	Type           string                       `json:"type"`
}

type RawRib struct {
	SubType        string                  `json:"sub_type"`
	SequenceNumber uint32                  `json:"sequence_number"`
	Prefix         bgp.AddrPrefixInterface `json:"prefix"`
	Entries        []*RawRibEntry          `json:"entries"`
	RouteFamily    bgp.RouteFamily         `json:"route_family"`
}

func raw(conf *MRT2JsonGlobalConf, f *os.File) {
	zmrt.MrtRawIterate(conf.InputFilePath, func(msg *mrt.MRTMessage) {
		if msg.Header.Type != mrt.TABLE_DUMPv2 {
			log.Fatal("not an MRT TABLE_DUMPv2")
		}
		switch mrt.MRTSubTypeTableDumpv2(msg.Header.SubType) {
		case mrt.PEER_INDEX_TABLE:
			peerIndexTable := msg.Body.(*mrt.PeerIndexTable)
			var out RawPeerIndexTableJSON
			out.CollectorBgpId = peerIndexTable.CollectorBgpId.String()
			out.ViewName = peerIndexTable.ViewName
			out.Type = "peer_index_table"
			for _, peer := range peerIndexTable.Peers {
				var outPeer RawPeer
				//outPeer.Type = peer.Type
				outPeer.BgpId = peer.BgpId.String()
				outPeer.IpAddress = peer.IpAddress.String()
				outPeer.AS = peer.AS
				out.Peers = append(out.Peers, &outPeer)
			}
			json, err := json.Marshal(out)
			if err != nil {
				log.Fatal("unable to json marshal peer table")
			}
			f.WriteString(string(json))
			f.WriteString("\n")
		case mrt.RIB_IPV4_UNICAST, mrt.RIB_IPV6_UNICAST,
			mrt.RIB_IPV4_MULTICAST, mrt.RIB_IPV6_MULTICAST, mrt.RIB_GENERIC:
			//
			rib := msg.Body.(*mrt.Rib)
			var out RawRib
			out.SequenceNumber = rib.SequenceNumber
			out.Prefix = rib.Prefix
			out.RouteFamily = rib.RouteFamily
			for _, entry := range rib.Entries {
				var ribOut RawRibEntry
				ribOut.PeerIndex = entry.PeerIndex
				ribOut.OriginatedTime = entry.OriginatedTime
				ribOut.PathIdentifier = entry.PathIdentifier
				ribOut.PathAttributes = entry.PathAttributes
				out.Entries = append(out.Entries, &ribOut)
			}
			out.SubType = zmrt.MrtSubTypeToName(msg.Header.SubType)
			json, err := json.Marshal(out)
			if err != nil {
				log.Fatal("unable to json marshal peer table")
			}
			f.WriteString(string(json))
			f.WriteString("\n")
		case mrt.GEO_PEER_TABLE:
			//geopeers := msg.Body.(*mrt.GeoPeerTable)
		default:
			log.Fatalf("unsupported subType: %v", msg.Header.SubType)
		}
	})
}

func paths(conf *MRT2JsonGlobalConf, f *os.File) {
	zmrt.MrtPathIterate(conf.InputFilePath, func(msg *zmrt.RIBEntry) {
		json, _ := json.Marshal(msg)
		f.WriteString(string(json))
		f.WriteString("\n")
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
	var f *os.File
	if conf.OutputFilePath == "-" {
		f = os.Stdout
	} else {
		var err error
		f, err = os.OpenFile(conf.OutputFilePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal("unable to open metadata file:", err.Error())
		}
		defer f.Close()
	}
	if os.Args[1] == "raw" {
		raw(&conf, f)
	} else if os.Args[1] == "entries" {
		paths(&conf, f)
	} else {
		log.Fatal("invalid command")
	}
}
