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

package http

import (
	"net/http"
	"testing"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPRespCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

type respReqOpt struct {
	id         int
	statusCode int
	headers    []pair
	token      int
}

func buildRespReq(opt respReqOpt) []byte {
	builder := flatbuffers.NewBuilder(1024)

	hdrLen := len(opt.headers)
	var hdrVec flatbuffers.UOffsetT
	if hdrLen > 0 {
		hdrs := []flatbuffers.UOffsetT{}
		for _, v := range opt.headers {
			name := builder.CreateString(v.name)
			value := builder.CreateString(v.value)
			A6.TextEntryStart(builder)
			A6.TextEntryAddName(builder, name)
			A6.TextEntryAddValue(builder, value)
			te := A6.TextEntryEnd(builder)
			hdrs = append(hdrs, te)
		}
		size := len(hdrs)
		hrc.ReqStartHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := hdrs[i]
			builder.PrependUOffsetT(te)
		}
		hdrVec = builder.EndVector(size)
	}

	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, uint32(opt.id))
	hrc.ReqAddConfToken(builder, uint32(opt.token))

	if opt.statusCode != 0 {
		hrc.ReqAddStatus(builder, uint16(opt.statusCode))
	}
	if hdrVec > 0 {
		hrc.ReqAddHeaders(builder, hdrVec)
	}
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	return builder.FinishedBytes()
}

func TestResponse_ID(t *testing.T) {
	out := buildRespReq(respReqOpt{id: 1234})
	r := CreateResponse(out)
	assert.Equal(t, 1234, int(r.ID()))
	ReuseResponse(r)
}

func TestResponse_ConfToken(t *testing.T) {
	out := buildRespReq(respReqOpt{token: 1234})
	r := CreateResponse(out)
	assert.Equal(t, 1234, int(r.ConfToken()))
	ReuseResponse(r)
}

func TestResponse_StatusCode(t *testing.T) {
	out := buildRespReq(respReqOpt{statusCode: 200})
	r := CreateResponse(out)
	assert.Equal(t, 200, r.StatusCode())
	ReuseResponse(r)
}

func TestResponse_WriteHeader(t *testing.T) {
	out := buildRespReq(respReqOpt{statusCode: 200})
	r := CreateResponse(out)

	r.WriteHeader(304)
	assert.Equal(t, 304, r.StatusCode())

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(builder))
	resp := hrc.GetRootAsResp(builder.FinishedBytes(), 0)
	assert.Equal(t, 304, int(resp.Status()))
	ReuseResponse(r)
}

func TestResponse_TwiceWriteHeader(t *testing.T) {
	out := buildRespReq(respReqOpt{statusCode: 200})
	r := CreateResponse(out)

	r.WriteHeader(304)
	r.WriteHeader(502)

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(builder))
	resp := hrc.GetRootAsResp(builder.FinishedBytes(), 0)
	assert.Equal(t, 304, int(resp.Status()))
	ReuseResponse(r)
}

func TestResponse_Header(t *testing.T) {
	out := buildRespReq(respReqOpt{headers: []pair{
		{"k", "v"},
		{"cache-control", "no-cache"},
		{"cache-control", "no-store"},
		{"cat", "dog"},
	}})
	r := CreateResponse(out)
	hdr := r.Header()
	assert.Equal(t, "v", hdr.Get("k"))
	assert.Equal(t, "no-cache", hdr.Get("Cache-Control"))
	assert.Equal(t, "no-cache", hdr.Get("cache-control"))

	hdr.Del("empty")
	hdr.Del("k")
	assert.Equal(t, "", hdr.Get("k"))

	hdr.Set("cache-control", "max-age=10s")
	assert.Equal(t, "max-age=10s", hdr.Get("Cache-Control"))
	hdr.Del("cache-Control")
	assert.Equal(t, "", hdr.Get("cache-control"))

	hdr.Set("k", "v2")
	hdr.Del("cat")

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(builder))
	resp := hrc.GetRootAsResp(builder.FinishedBytes(), 0)
	assert.Equal(t, 3, resp.HeadersLength())

	exp := http.Header{}
	exp.Set("Cache-Control", "")
	exp.Set("cat", "")
	exp.Set("k", "v2")
	res := http.Header{}
	for i := 0; i < resp.HeadersLength(); i++ {
		e := &A6.TextEntry{}
		resp.Headers(e, i)
		res.Add(string(e.Name()), string(e.Value()))
	}
	assert.Equal(t, exp, res)
	ReuseResponse(r)
}

func TestResponse_Write(t *testing.T) {
	out := buildRespReq(respReqOpt{
		id:         1234,
		statusCode: 200,
		headers:    []pair{{"k", "v"}},
	})
	r := CreateResponse(out)
	r.Write([]byte("hello "))
	r.Write([]byte("world"))

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(builder))
	resp := hrc.GetRootAsResp(builder.FinishedBytes(), 0)
	assert.Equal(t, 1234, int(resp.Id()))
	assert.Equal(t, 0, int(resp.Status()))
	assert.Equal(t, 0, resp.HeadersLength())
	assert.Equal(t, []byte("hello world"), resp.BodyBytes())
	ReuseResponse(r)
}
