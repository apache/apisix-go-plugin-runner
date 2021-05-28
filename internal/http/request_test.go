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
package http

import (
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func getRewriteAction(t *testing.T, b *flatbuffers.Builder) *hrc.Rewrite {
	buf := b.FinishedBytes()
	res := hrc.GetRootAsResp(buf, 0)
	tab := &flatbuffers.Table{}
	if res.Action(tab) {
		assert.Equal(t, hrc.ActionRewrite, res.ActionType())
		rewrite := &hrc.Rewrite{}
		rewrite.Init(tab.Bytes, tab.Pos)
		return rewrite
	}
	return nil
}

type pair struct {
	name  string
	value string
}

type reqOpt struct {
	srcIP   []byte
	method  A6.Method
	path    string
	headers []pair
	args    []pair
}

func buildReq(opt reqOpt) []byte {
	builder := flatbuffers.NewBuilder(1024)

	var ip flatbuffers.UOffsetT
	if len(opt.srcIP) > 0 {
		ip = builder.CreateByteVector(opt.srcIP)
	}

	var path flatbuffers.UOffsetT
	if opt.path != "" {
		path = builder.CreateString(opt.path)
	}

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
		hrc.RewriteStartHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := hdrs[i]
			builder.PrependUOffsetT(te)
		}
		hdrVec = builder.EndVector(size)
	}

	argsLen := len(opt.args)
	var argsVec flatbuffers.UOffsetT
	if argsLen > 0 {
		args := []flatbuffers.UOffsetT{}
		for _, v := range opt.args {
			name := builder.CreateString(v.name)
			value := builder.CreateString(v.value)
			A6.TextEntryStart(builder)
			A6.TextEntryAddName(builder, name)
			A6.TextEntryAddValue(builder, value)
			te := A6.TextEntryEnd(builder)
			args = append(args, te)
		}
		size := len(args)
		hrc.RewriteStartArgsVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := args[i]
			builder.PrependUOffsetT(te)
		}
		argsVec = builder.EndVector(size)
	}

	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	if ip > 0 {
		hrc.ReqAddSrcIp(builder, ip)
	}
	if opt.method != 0 {
		hrc.ReqAddMethod(builder, opt.method)
	}
	if path > 0 {
		hrc.ReqAddPath(builder, path)
	}
	if hdrVec > 0 {
		hrc.ReqAddHeaders(builder, hdrVec)
	}
	if argsVec > 0 {
		hrc.ReqAddArgs(builder, argsVec)
	}
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	return builder.FinishedBytes()
}

func TestSrcIp(t *testing.T) {
	for _, ip := range []net.IP{
		net.IPv4(127, 0, 0, 1),
		net.IPv4(127, 2, 3, 1),
		net.ParseIP("2001:db8::68"),
		net.ParseIP("::12"),
	} {
		out := buildReq(reqOpt{srcIP: ip})
		r := CreateRequest(out)
		assert.Equal(t, ip, r.SrcIP())
		ReuseRequest(r)
	}
}

func TestMethod(t *testing.T) {
	for _, m := range []A6.Method{
		A6.MethodGET,
		A6.MethodPATCH,
	} {
		out := buildReq(reqOpt{method: m})
		r := CreateRequest(out)
		assert.Equal(t, m.String(), r.Method())
	}
}

func TestPath(t *testing.T) {
	out := buildReq(reqOpt{path: "/apisix"})
	r := CreateRequest(out)
	assert.Equal(t, "/apisix", string(r.Path()))

	r.SetPath([]byte("/go"))
	assert.Equal(t, "/go", string(r.Path()))

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(1, builder))
	rewrite := getRewriteAction(t, builder)
	assert.Equal(t, "/go", string(rewrite.Path()))
}

func TestHeader(t *testing.T) {
	out := buildReq(reqOpt{headers: []pair{
		{"k", "v"},
		{"cache-control", "no-cache"},
		{"cache-control", "no-store"},
		{"cat", "dog"},
	}})
	r := CreateRequest(out)
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
	assert.True(t, r.FetchChanges(1, builder))
	rewrite := getRewriteAction(t, builder)
	assert.Equal(t, 3, rewrite.HeadersLength())

	exp := http.Header{}
	exp.Set("Cache-Control", "")
	exp.Set("cat", "")
	exp.Set("k", "v2")
	res := http.Header{}
	for i := 0; i < rewrite.HeadersLength(); i++ {
		e := &A6.TextEntry{}
		rewrite.Headers(e, i)
		res.Add(string(e.Name()), string(e.Value()))
	}
	assert.Equal(t, exp, res)
}

func TestArgs(t *testing.T) {
	out := buildReq(reqOpt{args: []pair{
		{"del", "a"},
		{"override", "a"},
		{"add", "a"},
	}})
	r := CreateRequest(out)
	args := r.Args()
	args.Add("add", "b")
	args.Set("set", "a")
	args.Set("override", "b")
	args.Del("del")

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(1, builder))
	rewrite := getRewriteAction(t, builder)

	exp := url.Values{}
	exp.Set("set", "a")
	exp.Set("override", "b")
	exp.Add("add", "a")
	exp.Add("add", "b")
	deleted := ""
	res := url.Values{}
	for i := 0; i < rewrite.ArgsLength(); i++ {
		e := &A6.TextEntry{}
		rewrite.Args(e, i)
		if e.Value() == nil {
			deleted = string(e.Name())
		} else {
			res.Add(string(e.Name()), string(e.Value()))
		}
	}
	assert.Equal(t, exp, res)
	assert.Equal(t, "del", deleted)
}
