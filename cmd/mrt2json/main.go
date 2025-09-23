/*
 * ZAnnotate Copyright 2025 Regents of the University of Michigan
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/osrg/gobgp/v3/pkg/packet/bgp"
	"github.com/osrg/gobgp/v3/pkg/packet/mrt"
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
	RouteFamily    bgp.Family              `json:"route_family"`
}

func raw(conf *MRT2JsonGlobalConf, f *os.File) {
	inputFile, err := os.Open(conf.InputFilePath)
	if err != nil {
		log.Fatal(err)
	}
	err = zmrt.MrtRawIterate(inputFile, func(msg *mrt.MRTMessage) error {
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
				outPeer.BgpId = peer.BgpId.String()
				outPeer.IpAddress = peer.IpAddress.String()
				outPeer.AS = peer.AS
				out.Peers = append(out.Peers, &outPeer)
			}
			json, err := json.Marshal(out)
			if err != nil {
				log.Fatal("unable to json marshal peer table")
			}
			if _, err := f.WriteString(string(json) + "\n"); err != nil {
				log.Fatalf("error writing to output file: %v", err)
			}
		case mrt.RIB_IPV4_UNICAST, mrt.RIB_IPV6_UNICAST,
			mrt.RIB_IPV4_MULTICAST, mrt.RIB_IPV6_MULTICAST, mrt.RIB_GENERIC,
			mrt.RIB_IPV4_MULTICAST_ADDPATH, mrt.RIB_IPV4_UNICAST_ADDPATH, mrt.RIB_IPV6_MULTICAST_ADDPATH,
			mrt.RIB_IPV6_UNICAST_ADDPATH:
			rib := msg.Body.(*mrt.Rib)
			var out RawRib
			out.SequenceNumber = rib.SequenceNumber
			out.Prefix = rib.Prefix
			out.RouteFamily = rib.Family
			for _, entry := range rib.Entries {
				var ribOut RawRibEntry
				ribOut.PeerIndex = entry.PeerIndex
				ribOut.OriginatedTime = entry.OriginatedTime
				ribOut.PathIdentifier = entry.PathIdentifier
				ribOut.PathAttributes = entry.PathAttributes
				out.Entries = append(out.Entries, &ribOut)
			}
			out.SubType = zmrt.MrtSubTypeToName(msg.Header.SubType)
			jsonData, err := json.Marshal(out)
			if err != nil {
				log.Fatal("unable to json marshal peer table")
			}
			if _, err = f.WriteString(string(jsonData) + "\n"); err != nil {
				log.Fatalf("error writing to output file: %v", err)
			}
		default:
			log.Fatalf("unsupported subType: %v", msg.Header.SubType)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error processing input file '%s': %v", conf.InputFilePath, err)
	}
}

func paths(conf *MRT2JsonGlobalConf, f *os.File) {
	inputFile, err := os.Open(conf.InputFilePath)
	if err != nil {
		log.Fatal(err)
	}
	err = zmrt.MrtPathIterate(inputFile, func(msg *zmrt.RIBEntry) error {
		var marshalledData []byte
		marshalledData, err = json.Marshal(msg)
		if _, err := f.WriteString(string(marshalledData) + "\n"); err != nil {
			return fmt.Errorf("error writing to output file: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	var conf MRT2JsonGlobalConf
	flags := flag.NewFlagSet("flags", flag.ExitOnError)
	flags.StringVar(&conf.InputFilePath, "input-file", "", "path to MRT file")
	flags.StringVar(&conf.OutputFilePath, "output-file", "-", "where should JSON output be saved")
	flags.StringVar(&conf.LogFilePath, "log-file", "", "where should JSON output be saved")
	flags.IntVar(&conf.Verbosity, "verbosity", 3, "where should JSON output be saved")

	if len(os.Args) < 2 {
		log.Fatalf("No command provided. Must choose raw or entries")
	}
	if err := flags.Parse(os.Args[2:]); err != nil {
		log.Fatalf("failed to parse input flags: %v", err)
	}

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
			log.Fatal("unable to open output file: ", err.Error())
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Errorf("unable to close output file: %v", err)
			}
		}()
	}
	switch os.Args[1] {
	case "raw":
		raw(&conf, f)
	case "entries":
		paths(&conf, f)
	default:
		log.Fatal("invalid command")
	}
}
