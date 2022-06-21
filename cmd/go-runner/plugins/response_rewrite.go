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

package plugins

import (
	"encoding/json"
	"net/http"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
)

func init() {
	err := plugin.RegisterPlugin(&ResponseRewrite{})
	if err != nil {
		log.Fatalf("failed to register plugin response-rewrite: %s", err)
	}
}

// ResponseRewrite is a demo to show how to rewrite response data.
type ResponseRewrite struct {
}

type ResponseRewriteConf struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (p *ResponseRewrite) Name() string {
	return "response-rewrite"
}

func (p *ResponseRewrite) ParseConf(in []byte) (interface{}, error) {
	conf := ResponseRewriteConf{}
	err := json.Unmarshal(in, &conf)
	return conf, err
}

func (p *ResponseRewrite) RequestFilter(interface{}, http.ResponseWriter, pkgHTTP.Request) {
}

func (p *ResponseRewrite) ResponseFilter(conf interface{}, w pkgHTTP.Response) {
	cfg := conf.(ResponseRewriteConf)
	if cfg.Status > 0 {
		w.WriteHeader(200)
	}

	w.Header().Set("X-Resp-A6-Runner", "Go")
	if len(cfg.Headers) > 0 {
		for k, v := range cfg.Headers {
			w.Header().Set(k, v)
		}
	}

	if len(cfg.Body) == 0 {
		return
	}
	_, err := w.Write([]byte(cfg.Body))
	if err != nil {
		log.Errorf("failed to write: %s", err)
	}
}
