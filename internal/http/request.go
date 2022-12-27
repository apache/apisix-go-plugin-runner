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
	"context"
	"encoding/binary"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/api7/ext-plugin-proto/go/A6"
	ei "github.com/api7/ext-plugin-proto/go/A6/ExtraInfo"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/common"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

type Request struct {
	// the root of the flatbuffers HTTPReqCall Request msg
	r *hrc.Req

	conn            net.Conn
	extraInfoHeader []byte

	path []byte

	hdr    *Header
	rawHdr http.Header

	args    url.Values
	rawArgs url.Values

	vars map[string][]byte
	body []byte

	ctx    context.Context
	cancel context.CancelFunc

	respHdr http.Header
}

func (r *Request) ConfToken() uint32 {
	return r.r.ConfToken()
}

func (r *Request) ID() uint32 {
	return r.r.Id()
}

func (r *Request) SrcIP() net.IP {
	return r.r.SrcIpBytes()
}

func (r *Request) Method() string {
	return r.r.Method().String()
}

func (r *Request) Path() []byte {
	if r.path == nil {
		return r.r.Path()
	}
	return r.path
}

func (r *Request) SetPath(path []byte) {
	r.path = path
}

func (r *Request) Header() pkgHTTP.Header {
	if r.hdr == nil {
		hdr := newHeader()
		hh := hdr.View()
		size := r.r.HeadersLength()
		obj := A6.TextEntry{}
		for i := 0; i < size; i++ {
			if r.r.Headers(&obj, i) {
				hh.Add(string(obj.Name()), string(obj.Value()))
			}
		}
		r.hdr = hdr
		r.rawHdr = hdr.Clone()
	}
	return r.hdr
}

func (r *Request) RespHeader() http.Header {
	if r.respHdr == nil {
		r.respHdr = http.Header{}
	}
	return r.respHdr
}

func cloneUrlValues(oldV url.Values) url.Values {
	nv := 0
	for _, vv := range oldV {
		nv += len(vv)
	}
	sv := make([]string, nv)
	newV := make(url.Values, len(oldV))
	for k, vv := range oldV {
		n := copy(sv, vv)
		newV[k] = sv[:n:n]
		sv = sv[n:]
	}
	return newV
}

func (r *Request) Args() url.Values {
	if r.args == nil {
		args := url.Values{}
		size := r.r.ArgsLength()
		obj := A6.TextEntry{}
		for i := 0; i < size; i++ {
			if r.r.Args(&obj, i) {
				args.Add(string(obj.Name()), string(obj.Value()))
			}
		}
		r.args = args
		r.rawArgs = cloneUrlValues(args)
	}
	return r.args
}

func (r *Request) Var(name string) ([]byte, error) {
	if r.vars == nil {
		r.vars = map[string][]byte{}
	}

	var v []byte
	var found bool

	if v, found = r.vars[name]; !found {
		var err error

		builder := util.GetBuilder()
		varName := builder.CreateString(name)
		ei.VarStart(builder)
		ei.VarAddName(builder, varName)
		varInfo := ei.VarEnd(builder)
		v, err = r.askExtraInfo(builder, ei.InfoVar, varInfo)
		util.PutBuilder(builder)

		if err != nil {
			return nil, err
		}

		r.vars[name] = v
	}
	return v, nil
}

func (r *Request) Body() ([]byte, error) {
	if len(r.body) > 0 {
		return r.body, nil
	}

	builder := util.GetBuilder()
	ei.ReqBodyStart(builder)
	bodyInfo := ei.ReqBodyEnd(builder)
	v, err := r.askExtraInfo(builder, ei.InfoReqBody, bodyInfo)
	if err != nil {
		return nil, err
	}

	r.body = v
	return v, nil
}

func (r *Request) Reset() {
	defer r.cancel()
	r.path = nil
	r.hdr = nil
	r.args = nil

	r.vars = nil
	r.body = nil
	r.conn = nil
	r.ctx = nil
	r.respHdr = nil
	// Keep the fields below
	// r.extraInfoHeader = nil
}

func (r *Request) FetchChanges(id uint32, builder *flatbuffers.Builder) bool {
	if r.path == nil && r.hdr == nil && r.args == nil && r.respHdr == nil {
		return false
	}

	var path flatbuffers.UOffsetT
	if r.path != nil {
		path = builder.CreateByteString(r.path)
	}

	var hdrVec, respHdrVec flatbuffers.UOffsetT
	if r.hdr != nil {
		hdrs := []flatbuffers.UOffsetT{}
		oldHdr := r.rawHdr
		newHdr := r.hdr.View()
		for n := range oldHdr {
			if _, ok := newHdr[n]; !ok {
				// deleted
				name := builder.CreateString(n)
				A6.TextEntryStart(builder)
				A6.TextEntryAddName(builder, name)
				te := A6.TextEntryEnd(builder)
				hdrs = append(hdrs, te)
			}
		}
		for n, v := range newHdr {
			if raw, ok := oldHdr[n]; !ok || raw[0] != v[0] {
				// set
				name := builder.CreateString(n)
				value := builder.CreateString(v[0])
				A6.TextEntryStart(builder)
				A6.TextEntryAddName(builder, name)
				A6.TextEntryAddValue(builder, value)
				te := A6.TextEntryEnd(builder)
				hdrs = append(hdrs, te)
			}
		}
		size := len(hdrs)
		hrc.RewriteStartHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := hdrs[i]
			builder.PrependUOffsetT(te)
		}
		hdrVec = builder.EndVector(size)
	}

	if r.respHdr != nil {
		respHdrs := []flatbuffers.UOffsetT{}
		for n, arr := range r.respHdr {
			for _, v := range arr {
				name := builder.CreateString(n)
				value := builder.CreateString(v)
				A6.TextEntryStart(builder)
				A6.TextEntryAddName(builder, name)
				A6.TextEntryAddValue(builder, value)
				te := A6.TextEntryEnd(builder)
				respHdrs = append(respHdrs, te)
			}
		}
		size := len(respHdrs)
		hrc.RewriteStartRespHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := respHdrs[i]
			builder.PrependUOffsetT(te)
		}
		respHdrVec = builder.EndVector(size)
	}

	var argsVec flatbuffers.UOffsetT
	if r.args != nil {
		args := []flatbuffers.UOffsetT{}
		oldArgs := r.rawArgs
		newArgs := r.args
		for n := range oldArgs {
			if _, ok := newArgs[n]; !ok {
				// deleted
				name := builder.CreateString(n)
				A6.TextEntryStart(builder)
				A6.TextEntryAddName(builder, name)
				te := A6.TextEntryEnd(builder)
				args = append(args, te)
			}
		}
		for n, v := range newArgs {
			if raw, ok := oldArgs[n]; !ok || !reflect.DeepEqual(raw, v) {
				// set / add
				for _, vv := range v {
					name := builder.CreateString(n)
					value := builder.CreateString(vv)
					A6.TextEntryStart(builder)
					A6.TextEntryAddName(builder, name)
					A6.TextEntryAddValue(builder, value)
					te := A6.TextEntryEnd(builder)
					args = append(args, te)
				}
			}
		}
		size := len(args)
		hrc.RewriteStartArgsVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := args[i]
			builder.PrependUOffsetT(te)
		}
		argsVec = builder.EndVector(size)
	}

	hrc.RewriteStart(builder)
	if path > 0 {
		hrc.RewriteAddPath(builder, path)
	}
	if hdrVec > 0 {
		hrc.RewriteAddHeaders(builder, hdrVec)
	}
	if respHdrVec > 0 {
		hrc.RewriteAddRespHeaders(builder, respHdrVec)
	}
	if argsVec > 0 {
		hrc.RewriteAddArgs(builder, argsVec)
	}
	rewrite := hrc.RewriteEnd(builder)

	hrc.RespStart(builder)
	hrc.RespAddId(builder, id)
	hrc.RespAddActionType(builder, hrc.ActionRewrite)
	hrc.RespAddAction(builder, rewrite)
	res := hrc.RespEnd(builder)
	builder.Finish(res)

	return true
}

func (r *Request) BindConn(c net.Conn) {
	r.conn = c
}

func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *Request) askExtraInfo(builder *flatbuffers.Builder,
	infoType ei.Info, info flatbuffers.UOffsetT) ([]byte, error) {

	ei.ReqStart(builder)
	ei.ReqAddInfoType(builder, infoType)
	ei.ReqAddInfo(builder, info)
	eiRes := ei.ReqEnd(builder)
	builder.Finish(eiRes)

	c := r.conn
	if len(r.extraInfoHeader) == 0 {
		r.extraInfoHeader = make([]byte, util.HeaderLen)
	}
	header := r.extraInfoHeader

	out := builder.FinishedBytes()
	size := len(out)
	binary.BigEndian.PutUint32(header, uint32(size))
	header[0] = util.RPCExtraInfo

	n, err := util.WriteBytes(c, header, len(header))
	if err != nil {
		util.WriteErr(n, err)
		return nil, common.ErrConnClosed
	}

	n, err = util.WriteBytes(c, out, size)
	if err != nil {
		util.WriteErr(n, err)
		return nil, common.ErrConnClosed
	}

	n, err = util.ReadBytes(c, header, util.HeaderLen)
	if util.ReadErr(n, err, util.HeaderLen) {
		return nil, common.ErrConnClosed
	}

	ty := header[0]
	header[0] = 0
	length := binary.BigEndian.Uint32(header)

	log.Infof("receive rpc type: %d data length: %d", ty, length)

	buf := make([]byte, length)
	n, err = util.ReadBytes(c, buf, int(length))
	if util.ReadErr(n, err, int(length)) {
		return nil, common.ErrConnClosed
	}

	resp := ei.GetRootAsResp(buf, 0)
	res := resp.ResultBytes()
	return res, nil
}

var reqPool = sync.Pool{
	New: func() interface{} {
		return &Request{}
	},
}

func CreateRequest(buf []byte) *Request {
	req := reqPool.Get().(*Request)
	req.r = hrc.GetRootAsReq(buf, 0)
	// because apisix has an implicit 60s timeout, so set the timeout to 56 seconds(smaller than 60s)
	// so plugin writer can still break the execution with a custom response before the apisix implicit timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 56*time.Second)
	req.ctx = ctx
	req.cancel = cancel
	return req
}

func ReuseRequest(r *Request) {
	r.Reset()
	reqPool.Put(r)
}
