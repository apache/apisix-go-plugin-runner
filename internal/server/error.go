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
package server

import (
	"github.com/ReneKroon/ttlcache/v2"
	A6Err "github.com/api7/ext-plugin-proto/go/A6/Err"
	flatbuffers "github.com/google/flatbuffers/go"
)

var (
	builder = flatbuffers.NewBuilder(256)
)

func ReportError(err error) []byte {
	builder.Reset()

	A6Err.RespStart(builder)

	var code A6Err.Code
	switch err {
	case ttlcache.ErrNotFound:
		code = A6Err.CodeCONF_TOKEN_NOT_FOUND
	default:
		code = A6Err.CodeSERVICE_UNAVAILABLE
	}

	A6Err.RespAddCode(builder, code)
	resp := A6Err.RespEnd(builder)
	builder.Finish(resp)
	return builder.FinishedBytes()
}
