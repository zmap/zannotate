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

	GeoIP2Conf GeoIP2Conf
	RoutingConf RoutingConf
	RDNSConf RDNSConf
}

type BasePluginConf struct {
	Threads int
	Enabled bool
}
