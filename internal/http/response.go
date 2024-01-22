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
	"encoding/binary"
	"net"
	"sync"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/common"
	ei "github.com/api7/ext-plugin-proto/go/A6/ExtraInfo"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPRespCall"
	flatbuffers "github.com/google/flatbuffers/go"
)

type Response struct {
	r *hrc.Req

	conn            net.Conn
	extraInfoHeader []byte

	hdr *Header

	statusCode int

	body *bytes.Buffer

	vars map[string][]byte
	// originBody is read-only
	originBody []byte
}

func (r *Response) askExtraInfo(builder *flatbuffers.Builder,
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
		r.hdr = newHeader(r.r)
	}

	return r.hdr
}

func (r *Response) Write(b []byte) (int, error) {
	if r.body == nil {
		r.body = &bytes.Buffer{}
	}

	return r.body.Write(b)
}

func (r *Response) Var(name string) ([]byte, error) {
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

func (r *Response) ReadBody() ([]byte, error) {
	if len(r.originBody) > 0 {
		return r.originBody, nil
	}

	builder := util.GetBuilder()
	ei.ReqBodyStart(builder)
	bodyInfo := ei.ReqBodyEnd(builder)
	v, err := r.askExtraInfo(builder, ei.InfoRespBody, bodyInfo)
	util.PutBuilder(builder)

	if err != nil {
		return nil, err
	}

	r.originBody = v
	return v, nil
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
		hdrVec = r.hdr.Build(builder)
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

func (r *Response) BindConn(c net.Conn) {
	r.conn = c
}

func (r *Response) Reset() {
	r.body = nil
	r.statusCode = 0
	r.hdr = nil
	r.conn = nil
	r.vars = nil
	r.originBody = nil
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
