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

type reqOpt struct {
	srcIP  []byte
	method A6.Method
	path   string
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
