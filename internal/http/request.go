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
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
)

type Request struct {
	// the root of the flatbuffers HTTPReqCall Request msg
	r *hrc.Req

	path []byte

	hdr    *Header
	rawHdr http.Header

	args    url.Values
	rawArgs url.Values
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

func (r *Request) Reset() {
	r.path = nil
	r.hdr = nil
	r.args = nil
}

func (r *Request) FetchChanges(id uint32, builder *flatbuffers.Builder) bool {
	if r.path == nil && r.hdr == nil && r.args == nil {
		return false
	}

	var path flatbuffers.UOffsetT
	if r.path != nil {
		path = builder.CreateByteString(r.path)
	}

	var hdrVec flatbuffers.UOffsetT
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

var reqPool = sync.Pool{
	New: func() interface{} {
		return &Request{}
	},
}

func CreateRequest(buf []byte) *Request {
	req := reqPool.Get().(*Request)
	req.r = hrc.GetRootAsReq(buf, 0)
	return req
}

func ReuseRequest(r *Request) {
	r.Reset()
	reqPool.Put(r)
}

type Header struct {
	http.Header
}

func newHeader() *Header {
	return &Header{
		Header: http.Header{},
	}
}

func (h *Header) View() http.Header {
	return h.Header
}
