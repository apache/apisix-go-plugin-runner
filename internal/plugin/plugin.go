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
package plugin

import (
	"net/http"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"

	inHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	"github.com/apache/apisix-go-plugin-runner/internal/util"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
)

func handle(conf RuleConf, w http.ResponseWriter, r pkgHTTP.Request) error {
	return nil
}

func reportAction(id uint32, req *inHTTP.Request, resp *inHTTP.Response) *flatbuffers.Builder {
	builder := util.GetBuilder()

	if resp != nil && resp.FetchChanges(id, builder) {
		return builder
	}

	hrc.RespStart(builder)
	hrc.RespAddId(builder, id)
	res := hrc.RespEnd(builder)
	builder.Finish(res)
	return builder
}

func HTTPReqCall(buf []byte) (*flatbuffers.Builder, error) {
	req := inHTTP.CreateRequest(buf)
	resp := inHTTP.CreateResponse()

	token := req.ConfToken()
	conf, err := GetRuleConf(token)
	if err != nil {
		return nil, err
	}
	err = handle(conf, resp, req)
	if err != nil {
		return nil, err
	}

	id := req.Id()
	builder := reportAction(id, req, resp)
	return builder, nil
}
