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

package plugin

import (
	"errors"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	A6 "github.com/api7/ext-plugin-proto/go/A6"
	pc "github.com/api7/ext-plugin-proto/go/A6/PrepareConf"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func TestPrepareConf(t *testing.T) {
	InitConfCache(10 * time.Millisecond)

	builder := flatbuffers.NewBuilder(1024)
	pc.ReqStart(builder)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	bd, _ := PrepareConf(b)
	out := bd.FinishedBytes()
	resp := pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(1), resp.ConfToken())

	bd, _ = PrepareConf(b)
	out = bd.FinishedBytes()
	resp = pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(2), resp.ConfToken())
}

func prepareConfWithData(builder *flatbuffers.Builder, arg ...flatbuffers.UOffsetT) {
	tes := []flatbuffers.UOffsetT{}
	for i := 0; i < len(arg); i += 2 {
		A6.TextEntryStart(builder)
		name := arg[i]
		value := arg[i+1]
		A6.TextEntryAddName(builder, name)
		A6.TextEntryAddValue(builder, value)
		te := A6.TextEntryEnd(builder)
		tes = append(tes, te)
	}

	pc.ReqStartConfVector(builder, len(tes))
	for i := len(tes) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(tes[i])
	}
	v := builder.EndVector(len(tes))

	pc.ReqStart(builder)
	pc.ReqAddConf(builder, v)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	PrepareConf(b)
}

func TestPrepareConfUnknownPlugin(t *testing.T) {
	InitConfCache(1 * time.Millisecond)
	builder := flatbuffers.NewBuilder(1024)

	name := builder.CreateString("xxx")
	value := builder.CreateString(`{"body":"yes"}`)
	prepareConfWithData(builder, name, value)
	res, _ := GetRuleConf(1)
	assert.Equal(t, 0, len(res))
}

func TestPrepareConfBadConf(t *testing.T) {
	InitConfCache(1 * time.Millisecond)
	builder := flatbuffers.NewBuilder(1024)

	f := func(in []byte) (conf interface{}, err error) {
		return nil, errors.New("ouch")
	}
	RegisterPlugin("bad_conf", f, emptyRequestFilter, emptyResponseFilter)
	name := builder.CreateString("bad_conf")
	value := builder.CreateString(`{"body":"yes"}`)
	prepareConfWithData(builder, name, value)
	res, _ := GetRuleConf(1)
	assert.Equal(t, 0, len(res))
}

func TestPrepareConfConcurrentlyWithoutKey(t *testing.T) {
	InitConfCache(10 * time.Millisecond)

	builder := flatbuffers.NewBuilder(1024)
	pc.ReqStart(builder)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	n := 10
	var wg sync.WaitGroup
	res := make([][]byte, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			bd, err := PrepareConf(b)
			assert.Nil(t, err)
			res[i] = bd.FinishedBytes()[:]
			wg.Done()
		}(i)
	}
	wg.Wait()

	tokens := make([]int, n)
	for i := 0; i < n; i++ {
		resp := pc.GetRootAsResp(res[i], 0)
		tokens[i] = int(resp.ConfToken())
	}

	sort.Ints(tokens)
	for i := 0; i < n; i++ {
		assert.Equal(t, i+1, tokens[i])
	}
}

func TestPrepareConfConcurrentlyWithTheSameKey(t *testing.T) {
	InitConfCache(10 * time.Millisecond)

	builder := flatbuffers.NewBuilder(1024)
	key := builder.CreateString("key")
	pc.ReqStart(builder)
	pc.ReqAddKey(builder, key)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	n := 10
	var wg sync.WaitGroup
	res := make([][]byte, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			bd, err := PrepareConf(b)
			assert.Nil(t, err)
			res[i] = bd.FinishedBytes()[:]
			wg.Done()
		}(i)
	}
	wg.Wait()

	tokens := make([]int, n)
	for i := 0; i < n; i++ {
		resp := pc.GetRootAsResp(res[i], 0)
		tokens[i] = int(resp.ConfToken())
	}

	sort.Ints(tokens)
	for i := 0; i < n; i++ {
		assert.Equal(t, 1, tokens[i])
	}
}

func TestPrepareConfConcurrentlyWithTheDifferentKey(t *testing.T) {
	InitConfCache(10 * time.Millisecond)

	builder := flatbuffers.NewBuilder(1024)
	n := 10
	var wg sync.WaitGroup
	var lock sync.Mutex
	res := make([][]byte, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			lock.Lock()
			key := builder.CreateString(strconv.Itoa(i))
			pc.ReqStart(builder)
			pc.ReqAddKey(builder, key)
			root := pc.ReqEnd(builder)
			builder.Finish(root)
			b := builder.FinishedBytes()
			lock.Unlock()

			bd, err := PrepareConf(b)
			assert.Nil(t, err)
			res[i] = bd.FinishedBytes()[:]
			wg.Done()
		}(i)
	}
	wg.Wait()

	tokens := make([]int, n)
	for i := 0; i < n; i++ {
		resp := pc.GetRootAsResp(res[i], 0)
		tokens[i] = int(resp.ConfToken())
	}

	sort.Ints(tokens)
	for i := 0; i < n; i++ {
		assert.Equal(t, i+1, tokens[i])
	}
}

func TestGetRuleConf(t *testing.T) {
	InitConfCache(1 * time.Millisecond)
	builder := flatbuffers.NewBuilder(1024)
	pc.ReqStart(builder)
	root := pc.ReqEnd(builder)
	builder.Finish(root)
	b := builder.FinishedBytes()

	bd, _ := PrepareConf(b)
	out := bd.FinishedBytes()
	resp := pc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(1), resp.ConfToken())

	res, _ := GetRuleConf(1)
	assert.Equal(t, 0, len(res))

	time.Sleep(2 * time.Millisecond)
	_, err := GetRuleConf(1)
	assert.Equal(t, ttlcache.ErrNotFound, err)
}

func TestGetRuleConfCheckConf(t *testing.T) {
	RegisterPlugin("echo", emptyParseConf, emptyRequestFilter, emptyResponseFilter)
	InitConfCache(1 * time.Millisecond)
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
	res, _ := GetRuleConf(1)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "echo", res[0].Name)
}
