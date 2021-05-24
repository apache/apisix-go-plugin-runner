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
package plugin

import (
	"testing"
	"time"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func TestHTTPReqCall(t *testing.T) {
	InitConfCache(10 * time.Millisecond)
	SetRuleConf(1, RuleConf{})

	builder := flatbuffers.NewBuilder(1024)
	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	b, err := HTTPReqCall(out)
	assert.Nil(t, err)

	out = b.FinishedBytes()
	resp := hrc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(233), resp.Id())
	assert.Equal(t, hrc.ActionNONE, resp.ActionType())
}
