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

	"github.com/ReneKroon/ttlcache/v2"
	A6 "github.com/api7/ext-plugin-proto/go/A6"
	pc "github.com/api7/ext-plugin-proto/go/A6/PrepareConf"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func TestPrepareConf(t *testing.T) {
	InitCache(10 * time.Millisecond)

	builder := flatbuffers.NewBuilder(1024)
	pc.ReqStart(builder)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	out := PrepareConf(b)
	resp := pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(1), resp.ConfToken())

	out = PrepareConf(b)
	resp = pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(2), resp.ConfToken())
}

func TestGetRuleConf(t *testing.T) {
	InitCache(1 * time.Millisecond)
	builder := flatbuffers.NewBuilder(1024)
	pc.ReqStart(builder)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	out := PrepareConf(b)
	resp := pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(3), resp.ConfToken())

	res, _ := GetRuleConf(3)
	assert.Equal(t, 0, len(res))

	time.Sleep(2 * time.Millisecond)
	_, err := GetRuleConf(3)
	assert.Equal(t, ttlcache.ErrNotFound, err)
}

func TestGetRuleConfCheckConf(t *testing.T) {
	InitCache(1 * time.Millisecond)
	builder := flatbuffers.NewBuilder(1024)

	name := builder.CreateString("echo")
	value := builder.CreateString(`{"body":"yes"}`)
	A6.TextEntryStart(builder)
	A6.TextEntryAddName(builder, name)
	A6.TextEntryAddValue(builder, value)
	te := A6.TextEntryEnd(builder)

	pc.ReqStartConfVector(builder, 1)
	builder.PrependUOffsetT(te)
	v := builder.EndVector(1)

	pc.ReqStart(builder)
	pc.ReqAddConf(builder, v)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	PrepareConf(b)
	res, _ := GetRuleConf(4)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "echo", res[0].Name)
}
