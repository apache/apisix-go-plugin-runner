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
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/api7/ext-plugin-proto/go/A6"
	ei "github.com/api7/ext-plugin-proto/go/A6/ExtraInfo"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/common"
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

func getVarInfo(t *testing.T, req *ei.Req) *ei.Var {
	tab := &flatbuffers.Table{}
	if req.Info(tab) {
		assert.Equal(t, ei.InfoVar, req.InfoType())
		info := &ei.Var{}
		info.Init(tab.Bytes, tab.Pos)
		return info
	}
	return nil
}

type pair struct {
	name  string
	value string
}

type reqOpt struct {
	srcIP      []byte
	method     A6.Method
	path       string
	headers    []pair
	respHeader []pair
	args       []pair
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
	var hdrVec, respHdrVec flatbuffers.UOffsetT
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

	if len(opt.respHeader) > 0 {
		respHdrs := []flatbuffers.UOffsetT{}
		for _, v := range opt.headers {
			name := builder.CreateString(v.name)
			value := builder.CreateString(v.value)
			A6.TextEntryStart(builder)
			A6.TextEntryAddName(builder, name)
			A6.TextEntryAddValue(builder, value)
			te := A6.TextEntryEnd(builder)
			respHdrs = append(respHdrs, te)
		}
		size := len(respHdrs)
		hrc.RewriteStartRespHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := respHdrs[i]
			builder.PrependUOffsetT(te)
		}
		respHdrVec = builder.EndVector(size)
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
	if respHdrVec > 0 {
		hrc.RewriteAddRespHeaders(builder, respHdrVec)
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

func TestVar(t *testing.T) {
	out := buildReq(reqOpt{})
	r := CreateRequest(out)

	cc, sc := net.Pipe()
	r.BindConn(cc)

	go func() {
		header := make([]byte, util.HeaderLen)
		n, err := util.ReadBytes(sc, header, util.HeaderLen)
		if util.ReadErr(n, err, util.HeaderLen) {
			return
		}

		ty := header[0]
		assert.Equal(t, byte(util.RPCExtraInfo), ty)
		header[0] = 0
		length := binary.BigEndian.Uint32(header)

		buf := make([]byte, length)
		n, err = util.ReadBytes(sc, buf, int(length))
		if util.ReadErr(n, err, int(length)) {
			return
		}

		req := ei.GetRootAsReq(buf, 0)
		info := getVarInfo(t, req)
		assert.Equal(t, "request_time", string(info.Name()))

		builder := util.GetBuilder()
		res := builder.CreateByteVector([]byte("1.0"))
		ei.RespStart(builder)
		ei.RespAddResult(builder, res)
		eiRes := ei.RespEnd(builder)
		builder.Finish(eiRes)
		out := builder.FinishedBytes()
		size := len(out)
		binary.BigEndian.PutUint32(header, uint32(size))
		header[0] = util.RPCExtraInfo

		n, err = util.WriteBytes(sc, header, len(header))
		if err != nil {
			util.WriteErr(n, err)
			return
		}

		n, err = util.WriteBytes(sc, out, size)
		if err != nil {
			util.WriteErr(n, err)
			return
		}
	}()

	for i := 0; i < 2; i++ {
		v, err := r.Var("request_time")
		assert.Nil(t, err)
		assert.Equal(t, "1.0", string(v))
	}
}

func TestVar_FailedToSendExtraInfoReq(t *testing.T) {
	out := buildReq(reqOpt{})
	r := CreateRequest(out)

	cc, sc := net.Pipe()
	r.BindConn(cc)

	go func() {
		header := make([]byte, util.HeaderLen)
		n, err := util.ReadBytes(sc, header, util.HeaderLen)
		if util.ReadErr(n, err, util.HeaderLen) {
			return
		}
		sc.Close()
	}()

	_, err := r.Var("request_time")
	assert.Equal(t, common.ErrConnClosed, err)
}

func TestVar_FailedToReadExtraInfoResp(t *testing.T) {
	out := buildReq(reqOpt{})
	r := CreateRequest(out)

	cc, sc := net.Pipe()
	r.BindConn(cc)

	go func() {
		header := make([]byte, util.HeaderLen)
		n, err := util.ReadBytes(sc, header, util.HeaderLen)
		if util.ReadErr(n, err, util.HeaderLen) {
			return
		}

		ty := header[0]
		assert.Equal(t, byte(util.RPCExtraInfo), ty)
		header[0] = 0
		length := binary.BigEndian.Uint32(header)

		buf := make([]byte, length)
		n, err = util.ReadBytes(sc, buf, int(length))
		if util.ReadErr(n, err, int(length)) {
			return
		}

		sc.Close()
	}()

	_, err := r.Var("request_time")
	assert.Equal(t, common.ErrConnClosed, err)
}

func TestContext(t *testing.T) {
	out := buildReq(reqOpt{})
	now := time.Now()
	timeout, _ := time.ParseDuration("56s")
	deadline := now.Add(timeout)
	r := CreateRequest(out)
	timer, ok := r.Context().Deadline()

	assert.True(t, ok)
	assert.True(t, timer.After(deadline))
	fmt.Println(ok, timer.After(deadline))
	ReuseRequest(r)
	assert.Equal(t, r.ctx, nil)
}

func TestRespHeader(t *testing.T) {
	out := buildReq(reqOpt{})
	r := CreateRequest(out)
	respHdr := r.RespHeader()

	respHdr.Set("resp-header", "this is resp-header")
	respHdr.Set("Set-Cookie", "mycookie=test")

	respHdr.Del("resp-header")

	builder := util.GetBuilder()
	assert.True(t, r.FetchChanges(1, builder))
	rewrite := getRewriteAction(t, builder)
	assert.Equal(t, 1, rewrite.RespHeadersLength())

	exp := http.Header{}
	exp.Set("Set-Cookie", "mycookie=test")
	res := http.Header{}
	for i := 0; i < rewrite.RespHeadersLength(); i++ {
		e := &A6.TextEntry{}
		rewrite.RespHeaders(e, i)
		res.Add(string(e.Name()), string(e.Value()))
	}
	assert.Equal(t, exp, res)
}

func TestBody(t *testing.T) {
	out := buildReq(reqOpt{})
	r := CreateRequest(out)

	cc, sc := net.Pipe()
	r.BindConn(cc)

	go func() {
		header := make([]byte, util.HeaderLen)
		n, err := util.ReadBytes(sc, header, util.HeaderLen)
		if util.ReadErr(n, err, util.HeaderLen) {
			return
		}

		ty := header[0]
		assert.Equal(t, byte(util.RPCExtraInfo), ty)
		header[0] = 0
		length := binary.BigEndian.Uint32(header)

		buf := make([]byte, length)
		n, err = util.ReadBytes(sc, buf, int(length))
		if util.ReadErr(n, err, int(length)) {
			return
		}

		req := ei.GetRootAsReq(buf, 0)
		assert.Equal(t, ei.InfoReqBody, req.InfoType())

		builder := util.GetBuilder()
		res := builder.CreateByteVector([]byte("Hello, Go Runner"))
		ei.RespStart(builder)
		ei.RespAddResult(builder, res)
		eiRes := ei.RespEnd(builder)
		builder.Finish(eiRes)
		out := builder.FinishedBytes()
		size := len(out)
		binary.BigEndian.PutUint32(header, uint32(size))
		header[0] = util.RPCExtraInfo

		n, err = util.WriteBytes(sc, header, len(header))
		if err != nil {
			util.WriteErr(n, err)
			return
		}

		n, err = util.WriteBytes(sc, out, size)
		if err != nil {
			util.WriteErr(n, err)
			return
		}
	}()

	v, err := r.Body()
	assert.Nil(t, err)
	assert.Equal(t, "Hello, Go Runner", string(v))
}
