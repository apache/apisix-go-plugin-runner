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

package server

import (
	"fmt"

	"github.com/ReneKroon/ttlcache/v2"
	A6Err "github.com/api7/ext-plugin-proto/go/A6/Err"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
)

type UnknownType struct {
	ty byte
}

func (err UnknownType) Error() string {
	return fmt.Sprintf("unknown type %d", err.ty)
}

func ReportError(err error) *flatbuffers.Builder {
	builder := util.GetBuilder()
	A6Err.RespStart(builder)

	var code A6Err.Code
	switch err {
	case ttlcache.ErrNotFound:
		code = A6Err.CodeCONF_TOKEN_NOT_FOUND
	default:
		switch err.(type) {
		case UnknownType:
			code = A6Err.CodeBAD_REQUEST
		default:
			code = A6Err.CodeSERVICE_UNAVAILABLE
		}
	}

	A6Err.RespAddCode(builder, code)
	resp := A6Err.RespEnd(builder)
	builder.Finish(resp)
	return builder
}
