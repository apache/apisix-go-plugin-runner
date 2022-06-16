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
	"net/http"
	"sync"

	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
)

type ReqResponse struct {
	hdr  http.Header
	body *bytes.Buffer
	code int
}

func (r *ReqResponse) Header() http.Header {
	if r.hdr == nil {
		r.hdr = http.Header{}
	}
	return r.hdr
}

func (r *ReqResponse) Write(b []byte) (int, error) {
	if r.body == nil {
		r.body = &bytes.Buffer{}
	}

	// APISIX will convert code 0 to 200, so we don't need to WriteHeader(http.StatusOK)
	// before writing the data
	return r.body.Write(b)
}

func (r *ReqResponse) WriteHeader(statusCode int) {
	if r.code != 0 {
		// official WriteHeader can't override written status
		// keep the same behavior
		return
	}
	r.code = statusCode
}

func (r *ReqResponse) Reset() {
	r.body = nil
	r.code = 0
	r.hdr = nil
}

func (r *ReqResponse) HasChange() bool {
	return !(r.body == nil && r.code == 0)
}

func (r *ReqResponse) FetchChanges(id uint32, builder *flatbuffers.Builder) bool {
	if !r.HasChange() {
		return false
	}

	hdrLen := len(r.hdr)
	var hdrVec flatbuffers.UOffsetT
	if hdrLen > 0 {
		hdrs := []flatbuffers.UOffsetT{}
		for n, arr := range r.hdr {
			for _, v := range arr {
				name := builder.CreateString(n)
				value := builder.CreateString(v)
				A6.TextEntryStart(builder)
				A6.TextEntryAddName(builder, name)
				A6.TextEntryAddValue(builder, value)
				te := A6.TextEntryEnd(builder)
				hdrs = append(hdrs, te)
			}
		}
		size := len(hdrs)
		hrc.StopStartHeadersVector(builder, size)
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

	hrc.StopStart(builder)
	if r.code == 0 {
		hrc.StopAddStatus(builder, 200)
	} else {
		hrc.StopAddStatus(builder, uint16(r.code))
	}
	if hdrLen > 0 {
		hrc.StopAddHeaders(builder, hdrVec)
	}
	if r.body != nil {
		hrc.StopAddBody(builder, bodyVec)
	}
	stop := hrc.StopEnd(builder)

	hrc.RespStart(builder)
	hrc.RespAddId(builder, id)
	hrc.RespAddActionType(builder, hrc.ActionStop)
	hrc.RespAddAction(builder, stop)
	res := hrc.RespEnd(builder)
	builder.Finish(res)

	return true
}

var reqRespPool = sync.Pool{
	New: func() interface{} {
		return &ReqResponse{}
	},
}

func CreateReqResponse() *ReqResponse {
	return reqRespPool.Get().(*ReqResponse)
}

func ReuseReqResponse(r *ReqResponse) {
	r.Reset()
	reqRespPool.Put(r)
}
