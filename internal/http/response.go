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
	"bytes"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPRespCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"net/http"
	"sync"
)

type Response struct {
	r *hrc.Req

	hdr    *Header
	rawHdr http.Header

	statusCode int

	body *bytes.Buffer
}

func (r *Response) ID() uint32 {
	return r.r.Id()
}

func (r *Response) StatusCode() int {
	if r.statusCode == 0 {
		return int(r.r.Status())
	}
	return r.statusCode
}

func (r *Response) Header() pkgHTTP.Header {
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

func (r *Response) Write(b []byte) (int, error) {
	if r.body == nil {
		r.body = &bytes.Buffer{}
	}

	return r.body.Write(b)
}

func (r *Response) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		// official WriteHeader can't override written status
		// keep the same behavior
		return
	}
	r.statusCode = statusCode
}

func (r *Response) ConfToken() uint32 {
	return r.r.ConfToken()
}

func (r *Response) HasChange() bool {
	return !(r.body == nil && r.hdr == nil && r.statusCode == 0)
}

func (r *Response) FetchChanges(builder *flatbuffers.Builder) bool {
	if !r.HasChange() {
		return false
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
		hrc.RespStartHeadersVector(builder, size)
		for i := size - 1; i >= 0; i-- {
			te := hdrs[i]
			builder.PrependUOffsetT(te)
		}
		hdrVec = builder.EndVector(size)
	}

	var bodyVec flatbuffers.UOffsetT
	if r.body != nil {
		b := r.body.Bytes()
		if len(b) > 0 {
			bodyVec = builder.CreateByteVector(b)
		}
	}

	hrc.RespStart(builder)
	if r.statusCode != 0 {
		hrc.RespAddStatus(builder, uint16(r.statusCode))
	}
	if hdrVec > 0 {
		hrc.RespAddHeaders(builder, hdrVec)
	}
	if bodyVec > 0 {
		hrc.RespAddBody(builder, bodyVec)
	}
	hrc.RespAddId(builder, r.r.Id())
	res := hrc.RespEnd(builder)
	builder.Finish(res)

	return true
}

func (r *Response) Reset() {
	r.body = nil
	r.statusCode = 0
	r.hdr = nil
}

var respPool = sync.Pool{
	New: func() interface{} {
		return &Response{}
	},
}

func CreateResponse(buf []byte) *Response {
	resp := respPool.Get().(*Response)

	resp.r = hrc.GetRootAsReq(buf, 0)
	return resp
}

func ReuseResponse(r *Response) {
	r.Reset()
	respPool.Put(r)
}
