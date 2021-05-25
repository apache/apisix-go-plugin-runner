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

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
)

type Request struct {
	// the root of the flatbuffers HTTPReqCall Request msg
	r *hrc.Req

	path []byte
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

func (r *Request) FetchChanges(id uint32, builder *flatbuffers.Builder) bool {
	if r.path == nil {
		return false
	}

	var path flatbuffers.UOffsetT
	if r.path != nil {
		path = builder.CreateByteString(r.path)
	}

	hrc.RewriteStart(builder)
	if path > 0 {
		hrc.RewriteAddPath(builder, path)
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

func CreateRequest(buf []byte) *Request {
	req := &Request{
		r: hrc.GetRootAsReq(buf, 0),
	}
	return req
}
