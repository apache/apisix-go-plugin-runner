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
	"io"
	"testing"
	"time"

	"github.com/apache/apisix-go-plugin-runner/internal/plugin"
	A6Err "github.com/api7/ext-plugin-proto/go/A6/Err"
	"github.com/stretchr/testify/assert"
)

func TestReportErrorCacheToken(t *testing.T) {
	plugin.InitConfCache(10 * time.Millisecond)

	_, err := plugin.GetRuleConf(uint32(999999))
	b := ReportError(err)
	out := b.FinishedBytes()
	resp := A6Err.GetRootAsResp(out, 0)
	assert.Equal(t, A6Err.CodeCONF_TOKEN_NOT_FOUND, resp.Code())
}

func TestReportErrorUnknownType(t *testing.T) {
	b := ReportError(UnknownType{23})
	out := b.FinishedBytes()
	resp := A6Err.GetRootAsResp(out, 0)
	assert.Equal(t, A6Err.CodeBAD_REQUEST, resp.Code())
}

func TestReportErrorUnknownErr(t *testing.T) {
	b := ReportError(io.EOF)
	out := b.FinishedBytes()
	resp := A6Err.GetRootAsResp(out, 0)
	assert.Equal(t, A6Err.CodeSERVICE_UNAVAILABLE, resp.Code())
}
