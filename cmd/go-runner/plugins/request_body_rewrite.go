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

const requestBodyRewriteName = "request-body-rewrite"

func init() {
	if err := plugin.RegisterPlugin(&RequestBodyRewrite{}); err != nil {
		log.Fatalf("failed to register plugin %s: %s", requestBodyRewriteName, err.Error())
	}
}

type RequestBodyRewrite struct {
	plugin.DefaultPlugin
}

type RequestBodyRewriteConfig struct {
	NewBody string `json:"new_body"`
}

func (*RequestBodyRewrite) Name() string {
	return requestBodyRewriteName
}

func (p *RequestBodyRewrite) ParseConf(in []byte) (interface{}, error) {
	conf := RequestBodyRewriteConfig{}
	err := json.Unmarshal(in, &conf)
	if err != nil {
		log.Errorf("failed to parse config for plugin %s: %s", p.Name(), err.Error())
	}
	return conf, err
}

func (*RequestBodyRewrite) RequestFilter(conf interface{}, _ http.ResponseWriter, r pkgHTTP.Request) {
	newBody := conf.(RequestBodyRewriteConfig).NewBody
	if newBody == "" {
		return
	}
	r.SetBody([]byte(newBody))
}
