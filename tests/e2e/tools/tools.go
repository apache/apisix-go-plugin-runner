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

package tools

import (
	"github.com/gavv/httpexpect/v2"
	"github.com/onsi/ginkgo"
	"net/http"
	"strings"
	"time"
)

var (
	token  = "edd1c9f034335f136f87ad84b625c8f1"
	A6Host = "http://127.0.0.1:9080"
)

func GetAdminToken() string {
	return token
}

func GetA6Expect() *httpexpect.Expect {
	t := ginkgo.GinkgoT()
	return httpexpect.New(t, A6Host)
}

type HttpTestCase struct {
	Object            *httpexpect.Expect
	Method            string
	Path              string
	Query             string
	Body              string
	Headers           map[string]string
	ExpectStatus      int
	ExpectStatusRange httpexpect.StatusRange
	ExpectCode        int
	ExpectBody        interface{}
	ExpectHeaders     map[string]string
	Sleep             time.Duration //ms
}

func RunTestCase(htc HttpTestCase) {
	var req *httpexpect.Request
	expect := htc.Object
	switch htc.Method {
	case http.MethodGet:
		req = expect.GET(htc.Path)
	case http.MethodPost:
		req = expect.POST(htc.Path)
	case http.MethodPut:
		req = expect.PUT(htc.Path)
	case http.MethodDelete:
		req = expect.DELETE(htc.Path)
	case http.MethodOptions:
		req = expect.OPTIONS(htc.Path)
	default:
	}

	if req == nil {
		panic("init request failed")
	}

	if htc.Sleep == 0 {
		time.Sleep(time.Duration(100) * time.Millisecond)
	} else {
		time.Sleep(htc.Sleep)
	}

	if len(htc.Query) > 0 {
		req.WithQueryString(htc.Query)
	}

	setContentType := false
	for hk, hv := range htc.Headers {
		req.WithHeader(hk, hv)
		if strings.ToLower(hk) == "content-type" {
			setContentType = true
		}
	}

	if !setContentType {
		req.WithHeader("Content-Type", "application/json")
	}

	if len(htc.Body) > 0 {
		req.WithText(htc.Body)
	}

	resp := req.Expect()

	if htc.ExpectStatus != 0 {
		resp.Status(htc.ExpectStatus)
	}

	if htc.ExpectStatusRange > 0 {
		resp.StatusRange(htc.ExpectStatusRange)
	}

	if htc.ExpectHeaders != nil {
		for hk, hv := range htc.ExpectHeaders {
			resp.Header(hk).Equal(hv)
		}
	}

	if htc.ExpectBody != nil {
		if body, ok := htc.ExpectBody.(string); ok {
			if len(body) == 0 {
				resp.Body().Empty()
			} else {
				resp.Body().Contains(body)
			}
		}

		if bodies, ok := htc.ExpectBody.([]string); ok && len(bodies) > 0 {
			for _, b := range bodies {
				resp.Body().Contains(b)
			}
		}
	}
}
