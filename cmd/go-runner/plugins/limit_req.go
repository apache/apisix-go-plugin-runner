// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package plugins

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
)

func init() {
	err := plugin.RegisterPlugin(&LimitReq{})
	if err != nil {
		log.Fatalf("failed to register plugin limit-req: %s", err)
	}
}

// LimitReq is a demo for a real world plugin
type LimitReq struct {
}

type LimitReqConf struct {
	Burst int     `json:"burst"`
	Rate  float64 `json:"rate"`

	limiter *rate.Limiter
}

func (p *LimitReq) Name() string {
	return "limit-req"
}

// ParseConf is called when the configuration is changed. And its output is unique per route.
func (p *LimitReq) ParseConf(in []byte) (interface{}, error) {
	conf := LimitReqConf{}
	err := json.Unmarshal(in, &conf)
	if err != nil {
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Limit(conf.Rate), conf.Burst)
	// the conf can be used to store route scope data
	conf.limiter = limiter
	return conf, nil
}

// Filter is called when a request hits the route
func (p *LimitReq) Filter(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
	li := conf.(LimitReqConf).limiter
	rs := li.Reserve()
	if !rs.OK() {
		// limit rate exceeded
		log.Infof("limit req rate exceeded")
		// stop filters with this response
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	time.Sleep(rs.Delay())
}
