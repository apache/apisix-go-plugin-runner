/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plugin

import (
	"net/http"

	"github.com/apache/apisix-go-plugin-runner/internal/plugin"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
)

// Plugin represents the Plugin
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// ParseConf is the method to parse given plugin configuration. When the
	// configuration can't be parsed, it will be skipped.
	ParseConf(in []byte) (conf interface{}, err error)

	// RequestFilter is the method to handle request.
	// It is like the `http.ServeHTTP`, plus the ctx and the configuration created by
	// ParseConf.
	//
	// When the `w` is written, the execution of plugin chain will be stopped.
	// We don't use onion model like Gin/Caddy because we don't serve the whole request lifecycle
	// inside the runner. The plugin is only a filter running at one stage.
	RequestFilter(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request)

	// ResponseFilter is the method to handle response.
	// This filter is currently only pre-defined and has not been implemented.
	ResponseFilter(conf interface{}, w pkgHTTP.Response)
}

// RegisterPlugin register a plugin. Plugin which has the same name can't be registered twice.
// This method should be called before calling `runner.Run`.
func RegisterPlugin(p Plugin) error {
	return plugin.RegisterPlugin(p.Name(), p.ParseConf, p.RequestFilter, p.ResponseFilter)
}

// DefaultPlugin provides the no-op implementation of the Plugin interface.
type DefaultPlugin struct{}

func (*DefaultPlugin) RequestFilter(interface{}, http.ResponseWriter, pkgHTTP.Request) {}
func (*DefaultPlugin) ResponseFilter(interface{}, pkgHTTP.Response)                    {}
