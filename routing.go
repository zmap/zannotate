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
	"github.com/zmap/zannotate/zrouting"
)


type RoutingConf struct {
	RoutingTablePath string
	ASNamesPath      string
	ASDataPath       string
}

type RoutingAnnotatorFactory struct {
	rlt *zrouting.RoutingLookupTree
}


type RoutingAnnotator struct {
	Factory *RoutingAnnotatorFactory
	rlt *zrouting.RoutingLookupTree
	Id int
}

// Routing Annotator Factory (Global)

func (a *RoutingAnnotatorFactory) MakeAnnotator(i int) *RoutingAnnotator {
	var v RoutingAnnotator
	v.Factory = a
	v.rlt = a.rlt
	v.Id = i
	return &v
}

func (a *RoutingAnnotatorFactory) Initialize(conf *GlobalConf) error {
	// Routing Lookup Trees are thread-safe
	a.rlt = new(zrouting.RoutingLookupTree)
	f, err := os.Open(conf.RoutingConf.RoutingTablePath)
	if err != nil {
		return err
	}
	if conf.RoutingConf.ASNamesPath != "" {
		f, err := os.Open(conf.RoutingConf.ASNamesPath)
		if err != nil {
			return err
		}
		a.rlt.PopulateASnames(f)
	}
	a.rlt.PopulateFromMRT(f)
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

