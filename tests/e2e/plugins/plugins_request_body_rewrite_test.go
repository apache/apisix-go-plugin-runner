package plugins_test

import (
	"net/http"

	"github.com/apache/apisix-go-plugin-runner/tests/e2e/tools"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
)

var _ = ginkgo.Describe("RequestBodyRewrite Plugin", func() {
	table.DescribeTable("tries to test request body rewrite feature",
		func(tc tools.HttpTestCase) {
			tools.RunTestCase(tc)
		},
		table.Entry("config APISIX", tools.HttpTestCase{
			Object: tools.GetA6CPExpect(),
			Method: http.MethodPut,
			Path:   "/apisix/admin/routes/1",
			Body: `{
				"uri":"/echo",
				"plugins":{
					"ext-plugin-pre-req":{
						"conf":[
							{
								"name":"request-body-rewrite",
								"value":"{\"new_body\":\"request body rewrite\"}"
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
		}),
		table.Entry("should rewrite request body", tools.HttpTestCase{
			Object:     tools.GetA6DPExpect(),
			Method:     http.MethodGet,
			Path:       "/echo",
			ExpectBody: "request body rewrite",
		}),
	)
})
