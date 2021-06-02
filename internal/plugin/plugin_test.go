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
	"testing"
	"time"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"

	inHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
)

var (
	emptyParseConf = func(in []byte) (conf interface{}, err error) {
		return string(in), nil
	}

	emptyFilter = func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		return
	}
)

func TestHTTPReqCall(t *testing.T) {
	InitConfCache(10 * time.Millisecond)
	SetRuleConf(1, RuleConf{})

	builder := flatbuffers.NewBuilder(1024)
	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	b, err := HTTPReqCall(out)
	assert.Nil(t, err)

	out = b.FinishedBytes()
	resp := hrc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(233), resp.Id())
	assert.Equal(t, hrc.ActionNONE, resp.ActionType())
}

func TestRegisterPlugin(t *testing.T) {
	assert.Equal(t, ErrMissingParseConfMethod,
		RegisterPlugin("bad_conf", nil, emptyFilter))
	assert.Equal(t, ErrMissingFilterMethod,
		RegisterPlugin("bad_conf", emptyParseConf, nil))
}

func TestFilter(t *testing.T) {
	fooParseConf := func(in []byte) (conf interface{}, err error) {
		return "foo", nil
	}
	fooFilter := func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		w.Header().Add("foo", "bar")
		assert.Equal(t, "foo", conf.(string))
	}
	barParseConf := func(in []byte) (conf interface{}, err error) {
		return "bar", nil
	}
	barFilter := func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		r.Header().Set("foo", "bar")
		assert.Equal(t, "bar", conf.(string))
	}

	RegisterPlugin("foo", fooParseConf, fooFilter)
	RegisterPlugin("bar", barParseConf, barFilter)

	builder := flatbuffers.NewBuilder(1024)
	fooName := builder.CreateString("foo")
	fooConf := builder.CreateString("foo")
	barName := builder.CreateString("bar")
	barConf := builder.CreateString("bar")
	prepareConfWithData(builder, fooName, fooConf, barName, barConf)

	res, _ := GetRuleConf(1)
	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	req := inHTTP.CreateRequest(out)
	resp := inHTTP.CreateResponse()
	filter(res, resp, req)

	assert.Equal(t, "bar", resp.Header().Get("foo"))
	assert.Equal(t, "", req.Header().Get("foo"))

	req = inHTTP.CreateRequest(out)
	resp = inHTTP.CreateResponse()
	prepareConfWithData(builder, barName, barConf, fooName, fooConf)
	res, _ = GetRuleConf(2)
	filter(res, resp, req)

	assert.Equal(t, "bar", resp.Header().Get("foo"))
	assert.Equal(t, "bar", req.Header().Get("foo"))
}
