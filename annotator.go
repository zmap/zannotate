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

package zannotate

import "net"


type Annotator interface {
	Initialize() error
	Annotate(ip net.IP) interface{}
	GetFieldName() string
	Close(c GlobalConf) error
}

type AnnotatorFactory interface {
	Initialize(c GlobalConf) error
	MakeAnnotator(i int) *Annotator
	Close() error
}

