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
	"errors"
	"math/rand"
	"net/http"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
)

const (
	plugin_name = "fault-injection"
)

func init() {
	err := plugin.RegisterPlugin(&FaultInjection{})
	if err != nil {
		log.Fatalf("failed to register plugin %s: %s", plugin_name, err)
	}
}

// FaultInjection is used in the benchmark
type FaultInjection struct {
}

type FaultInjectionConf struct {
	Body       string `json:"body"`
	HttpStatus int    `json:"http_status"`
	Percentage int    `json:"percentage"`
}

func (p *FaultInjection) Name() string {
	return plugin_name
}

func (p *FaultInjection) ParseConf(in []byte) (interface{}, error) {
	conf := FaultInjectionConf{Percentage: -1}
	err := json.Unmarshal(in, &conf)
	if err != nil {
		return nil, err
	}

	// schema check
	if conf.HttpStatus < 200 {
		return nil, errors.New("bad http_status")
	}
	if conf.Percentage == -1 {
		conf.Percentage = 100
	} else if conf.Percentage < 0 || conf.Percentage > 100 {
		return nil, errors.New("bad percentage")
	}

	return conf, err
}

func sampleHit(percentage int) bool {
	return rand.Intn(100) < percentage
}

func (p *FaultInjection) Filter(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
	fc := conf.(FaultInjectionConf)
	if !sampleHit(fc.Percentage) {
		return
	}

	w.WriteHeader(fc.HttpStatus)
	body := fc.Body
	if len(body) == 0 {
		return
	}

	_, err := w.Write([]byte(body))
	if err != nil {
		log.Errorf("failed to write: %s", err)
	}
}

func (p *FaultInjection) RespFilter(interface{}, pkgHTTP.Response) {

}
