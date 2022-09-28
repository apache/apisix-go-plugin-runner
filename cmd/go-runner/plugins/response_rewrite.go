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
	"bytes"
	"encoding/json"
	"fmt"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
	"regexp"
)

func init() {
	err := plugin.RegisterPlugin(&ResponseRewrite{})
	if err != nil {
		log.Fatalf("failed to register plugin response-rewrite: %s", err)
	}
}

type RegexFilter struct {
	Regex   string `json:"regex"`
	Scope   string `json:"scope"`
	Replace string `json:"replace"`

	regexComplied *regexp.Regexp
}

// ResponseRewrite is a demo to show how to rewrite response data.
type ResponseRewrite struct {
	// Embed the default plugin here,
	// so that we don't need to reimplement all the methods.
	plugin.DefaultPlugin
}

type ResponseRewriteConf struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Filters []RegexFilter     `json:"filters"`
}

func (p *ResponseRewrite) Name() string {
	return "response-rewrite"
}

func (p *ResponseRewrite) ParseConf(in []byte) (interface{}, error) {
	conf := ResponseRewriteConf{}
	err := json.Unmarshal(in, &conf)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(conf.Filters); i++ {
		if reg, err := regexp.Compile(conf.Filters[i].Regex); err != nil {
			return nil, fmt.Errorf("failed to compile regex `%s`: %v",
				conf.Filters[i].Regex, err)
		} else {
			conf.Filters[i].regexComplied = reg
		}
	}
	return conf, nil
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

	body := []byte(cfg.Body)
	if len(cfg.Filters) > 0 {
		originBody, err := w.ReadBody()
		if err != nil {
			log.Errorf("failed to read response body: ", err)
			return
		}
		matched := false
		for i := 0; i < len(cfg.Filters); i++ {
			f := cfg.Filters[i]
			found := f.regexComplied.Find(originBody)
			if found != nil {
				matched = true
				if f.Scope == "once" {
					originBody = bytes.Replace(originBody, found, []byte(f.Replace), 1)
				} else if f.Scope == "global" {
					originBody = bytes.ReplaceAll(originBody, found, []byte(f.Replace))
				}
			}
		}
		if matched == true {
			body = originBody
			goto write
		}
		// When configuring the Filters field, the Body field will be invalid.
		return
	}

	if len(cfg.Body) == 0 {
		return
	}
write:
	_, err := w.Write(body)
	if err != nil {
		log.Errorf("failed to write: %s", err)
	}
}
