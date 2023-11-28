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
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyRewrite_ParseConf(t *testing.T) {
	testCases := []struct {
		name    string
		in      []byte
		expect  string
		wantErr bool
	}{
		{
			"happy path",
			[]byte(`{"new_body":"hello"}`),
			"hello",
			false,
		},
		{
			"empty conf",
			[]byte(``),
			"",
			true,
		},
		{
			"empty body",
			[]byte(`{"new_body":""}`),
			"",
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := new(RequestBodyRewrite)
			conf, err := p.ParseConf(tc.in)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expect, conf.(RequestBodyRewriteConfig).NewBody)
		})
	}
}

func TestRequestBodyRewrite_RequestFilter(t *testing.T) {
	req := &mockHTTPRequest{body: []byte("hello")}
	p := new(RequestBodyRewrite)
	conf, err := p.ParseConf([]byte(`{"new_body":"See ya"}`))
	require.NoError(t, err)
	p.RequestFilter(conf, nil, req)
	require.Equal(t, []byte("See ya"), req.body)
}

// mockHTTPRequest implements pkgHTTP.Request
type mockHTTPRequest struct {
	body []byte
}

func (r *mockHTTPRequest) SetBody(body []byte) {
	r.body = body
}

func (*mockHTTPRequest) Args() url.Values {
	panic("unimplemented")
}

func (*mockHTTPRequest) Body() ([]byte, error) {
	panic("unimplemented")
}

func (*mockHTTPRequest) Context() context.Context {
	panic("unimplemented")
}

func (*mockHTTPRequest) Header() pkgHTTP.Header {
	panic("unimplemented")
}

func (*mockHTTPRequest) ID() uint32 {
	panic("unimplemented")
}

func (*mockHTTPRequest) Method() string {
	panic("unimplemented")
}

func (*mockHTTPRequest) Path() []byte {
	panic("unimplemented")
}

func (*mockHTTPRequest) RespHeader() http.Header {
	panic("unimplemented")
}

func (*mockHTTPRequest) SetPath([]byte) {
	panic("unimplemented")
}

func (*mockHTTPRequest) SrcIP() net.IP {
	panic("unimplemented")
}

func (*mockHTTPRequest) Var(string) ([]byte, error) {
	panic("unimplemented")
}
