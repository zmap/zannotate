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

package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/zmap/zannotate"
)

func main() {

	var conf zannotate.GlobalConf
	flags := flag.NewFlagSet("flags", flag.ExitOnError)
	flags.StringVar(&conf.InputFilePath, "input-file", "-", "ip addresses to read")
	flags.StringVar(&conf.InputFileType, "input-file-type", "ips", "ips, csv, json")
	flags.StringVar(&conf.OutputFilePath, "output-file", "-", "where should JSON output be saved")
	flags.StringVar(&conf.MetadataFilePath, "metadata-file", "",
		"where should JSON metadata be saved")
	flags.StringVar(&conf.LogFilePath, "log-file", "", "where should JSON logs be saved")
	flags.IntVar(&conf.Verbosity, "verbosity", 3, "log verbosity: 1 (lowest)--5 (highest)")
	// json annotation configuration
	flags.StringVar(&conf.JSONIPFieldName, "json-ip-field", "ip", "key in JSON that contains IP address")
	flags.StringVar(&conf.JSONAnnotationFieldName, "json-annotation-field", "zannotate", "key that metadata is injected at")

	// MaxMind GeoIP2
	flags.BoolVar(&conf.GeoIP2Conf.Enabled, "geoip2", false, "geolocate")
	flags.StringVar(&conf.GeoIP2Conf.Path, "geoip2-database", "",
		"path to MaxMind GeoIP2 database")
	flags.StringVar(&conf.GeoIP2Conf.Mode, "geoip2-mode", "mmap",
		"how to open database: mmap or memory")
	flags.StringVar(&conf.GeoIP2Conf.Language, "geoip2-language", "en",
		"how to open database: mmap or memory")
	flags.StringVar(&conf.GeoIP2Conf.RawInclude, "geoip2-fields", "*",
		"city, continent, country, location, postal, registered_country, subdivisions, traits")
	flags.IntVar(&conf.GeoIP2Conf.Threads, "geoip-threads", 5, "how many geoIP processing threads to use")

	// Routing Table AS Data
	flags.BoolVar(&conf.RoutingConf.Enabled, "routing", false, "routing")
	flags.StringVar(&conf.RoutingConf.RoutingTablePath, "mrt-file", "", "path to MRT TABLE_DUMPv2 file")
	flags.StringVar(&conf.RoutingConf.ASNamesPath, "as-names", "", "path to as names file")
	flags.IntVar(&conf.RoutingConf.Threads, "routing-threads", 5, "how many routing processing threads to use")

	// Reverse DNS Lookup
	flags.BoolVar(&conf.ReverseDNSConf.Enabled, "reverse-dns", false, "reverse dns lookup")
	flags.StringVar(&conf.ReverseDNSConf.RawResolvers, "dns-servers", "", "list of DNS servers to use for DNS lookups")
	flags.IntVar(&conf.ReverseDNSConf.Threads, "rdns-threads", 100, "how many reverse dns threads")

	flags.Parse(os.Args[1:])
	if conf.LogFilePath != "" {
		f, err := os.OpenFile(conf.LogFilePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Unable to open log file (%s): %s", conf.LogFilePath, err.Error())
		}
		log.SetOutput(f)
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
	// Check that we're doing anything
	if conf.GeoIP2 != true && conf.Routing != true {
		log.Fatal("No action requested")
	}
	if conf.InputFileType != "ips" && conf.InputFileType != "json" {
		log.Fatal("invalid input file type")
	}
	zannotate.DoAnnotation(&conf)
}
