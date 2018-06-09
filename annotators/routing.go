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

package zannotate

import (
	"os"
	"net"
	"errors"
	"flag"

	log "github.com/sirupsen/logrus"
	"github.com/zmap/zannotate/zrouting"
)

type RoutingAnnotatorFactory struct {
	BasePluginConf
	RoutingTablePath string
	ASNamesPath      string
	ASDataPath       string

	rlt *zrouting.RoutingLookupTree
}

type RoutingAnnotator struct {
	Factory *RoutingAnnotatorFactory
	rlt *zrouting.RoutingLookupTree
	Id int
}

// Routing Annotator Factory (Global)
func (a *RoutingAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&a.Enabled, "routing", false, "annotate with origin AS lookup")
	flags.StringVar(&a.RoutingTablePath, "routing-mrt-file", "",
		"path to MRT TABLE_DUMPv2 file")
	flags.StringVar(&a.ASNamesPath, "routing-as-names", "", "path to as names file")
	flags.IntVar(&a.Threads, "routing-threads", 5,
		"how many routing processing threads to use")
}

func (a *RoutingAnnotatorFactory) IsEnabled() bool {
	return a.Enabled
}

func (a *RoutingAnnotatorFactory) Initialize(conf *GlobalConf) error {
	if a.RoutingTablePath == "" {
		return errors.New("no routing file (MRT TABLE_DUMPv2) provided")
	}
	log.Info("will add routing using ", a.RoutingTablePath)
	// Routing Lookup Trees are thread-safe
	a.rlt = new(zrouting.RoutingLookupTree)
	f, err := os.Open(a.RoutingTablePath)
	if err != nil {
		return err
	}
	if a.ASNamesPath != "" {
		f, err := os.Open(a.ASNamesPath)
		if err != nil {
			return err
		}
		a.rlt.PopulateASnames(f)
	}
	a.rlt.PopulateFromMRT(f)
	return nil
}

func (a *RoutingAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v RoutingAnnotator
	v.Factory = a
	v.rlt = a.rlt
	v.Id = i
	return &v
}

func (a *RoutingAnnotatorFactory) Close() error {
	return nil
}

// Routing Annotator (Per-Worker)

func (a *RoutingAnnotator) Initialize() error {
	return nil
}

func (a *RoutingAnnotator) GetFieldName() string {
	return "routing"
}

func (a *RoutingAnnotator) Annotate(ip net.IP) interface{} {
	ret, err := a.rlt.Get(ip)
	if err != nil {
		return nil
	}
	return ret
}

func (a *RoutingAnnotator) Close() error {
	return nil
}

func init() {
	s := new(RoutingAnnotatorFactory)
	RegisterAnnotator(s)
}
