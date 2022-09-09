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

package plugins_test

import (
	"net/http"

	"github.com/apache/apisix-go-plugin-runner/tests/e2e/tools"
	"github.com/gavv/httpexpect/v2"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
)

var _ = ginkgo.Describe("Fault-injection plugin", func() {
	table.DescribeTable("tries to test fault-injection feature.",
		func(tc tools.HttpTestCase) {
			tools.RunTestCase(tc)
		},
		table.Entry("Config APISIX.", tools.HttpTestCase{
			Object: tools.GetA6CPExpect(),
			Method: http.MethodPut,
			Path:   "/apisix/admin/routes/1",
			Body: `{
				"uri":"/test/go/runner/faultinjection",
				"plugins":{
					"ext-plugin-pre-req":{
						"conf":[
							{
								"name":"fault-injection",
								"value":"{\"http_status\":400,\"body\":\"hello\"}"
							}
						]
					}
				},
				"upstream":{
					"nodes":{
						"web:8888":1
					},
					"type":"roundrobin"
				}
			}`,
			Headers:           map[string]string{"X-API-KEY": tools.GetAdminToken()},
			ExpectStatusRange: httpexpect.Status2xx,
		}),
		table.Entry("Test if fault-injection plugin work.", tools.HttpTestCase{
			Object:       tools.GetA6DPExpect(),
			Method:       http.MethodGet,
			Path:         "/test/go/runner/faultinjection",
			ExpectBody:   []string{"hello"},
			ExpectStatus: http.StatusBadRequest,
		}),
	)
})
