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

package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"

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
	flags.StringVar(&conf.InputIPFieldName, "input-ip-field", "ip", "key in JSON or column in CSV that contains IP address")
	flags.StringVar(&conf.OutputAnnotationFieldName, "output-annotation-field", "zannotate", "key that metadata is injected at, used for both CSV and JSON file inputs to preserve data in the input file")
	// encode/decode threads
	flags.IntVar(&conf.InputDecodeThreads, "input-decode-threads", 3, "number of golang processes to decode input data (e.g., json)")
	flags.IntVar(&conf.OutputEncodeThreads, "output-encode-threads", 3, "number of golang processes to encode output data (e.g., json)")
	prepareUsageString(flags) // Constructs the --help string
	// parse
	err := flags.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("failed to parse input flags: %v", err)
	}
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
	if !slices.Contains([]string{"ips", "json", "csv"}, conf.InputFileType) {
		log.Fatal("invalid input file type")
	}
	// check if we have any annotations to be performed
	annotationsPresent := false
	for _, annotator := range zannotate.Annotators {
		if annotator.IsEnabled() {
			annotationsPresent = true
			break
		}
	}
	if !annotationsPresent {
		log.Fatal("You must select at least one annotation to perform")
	}
	// perform sanity checks
	if conf.InputFileType == "ips" {
		conf.OutputAnnotationFieldName = ""
	}

	// perform annotation
	zannotate.DoAnnotation(&conf)
}

func prepareUsageString(flags *flag.FlagSet) {
	// Build grouped flag help: snapshot flags before each annotator registers its own.
	type flagGroup struct {
		name  string
		flags []string
	}
	snapshot := func() map[string]bool {
		seen := make(map[string]bool)
		flags.VisitAll(func(f *flag.Flag) { seen[f.Name] = true })
		return seen
	}
	groups := make([]flagGroup, 0, 1 + len(zannotate.Annotators)) // Global options + each annotator
	// Collect the global flags registered above.
	globalFlags := snapshot()
	globalNames := slices.Collect(maps.Keys(globalFlags))
	groups = append(groups, flagGroup{"Global", globalNames})
	// add the flags defined by each of the annotation modules
	for _, annotator := range zannotate.Annotators {
		pre := snapshot()
		annotator.AddFlags(flags)
		var added []string
		flags.VisitAll(func(f *flag.Flag) {
			if !pre[f.Name] {
				added = append(added, f.Name)
			}
		})
		if len(added) > 0 {
			groups = append(groups, flagGroup{annotator.GroupName(), added})
		}
	}
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [global options] <-module [module-options]>...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "At least one annotation module must be specified (e.g. -geoip2, -rdns, -routing).\n\n")
		for _, g := range groups {
			fmt.Fprintf(os.Stderr, "\n%s Options:\n", g.name)
			// Create a temporary FlagSet containing only this group's flags so we
			// can delegate formatting to PrintDefaults.
			tmp := flag.NewFlagSet("", flag.ContinueOnError)
			for _, name := range g.flags {
				tmp.Var(flags.Lookup(name).Value, name, flags.Lookup(name).Usage)
			}
			tmp.PrintDefaults()
		}
	}
}
