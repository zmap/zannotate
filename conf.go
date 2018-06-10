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
	"flag"
	"net"
)

// interfaces for annotation plugins

type Annotator interface {
	Initialize() error
	Annotate(ip net.IP) interface{}
	GetFieldName() string
	Close() error
}

type AnnotatorFactory interface {
	Initialize(c *GlobalConf) error
	AddFlags(flags *flag.FlagSet)
	GetWorkers() int
	IsEnabled() bool
	MakeAnnotator(i int) Annotator
	Close() error
}

type BasePluginConf struct {
	Threads int
	Enabled bool
}

// global library configuration

type GlobalConf struct {
	InputFilePath           string
	InputFileType           string
	OutputFilePath          string
	MetadataFilePath        string
	LogFilePath             string
	Verbosity               int
	Threads                 int
	JSONIPFieldName         string
	JSONAnnotationFieldName string
	InputDecodeThreads      int
	OutputEncodeThreads     int
}

var Annotators []AnnotatorFactory

func RegisterAnnotator(af AnnotatorFactory) {
	Annotators = append(Annotators, af)
}
