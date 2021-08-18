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
	"net/http"
	"testing"
	"time"

	inHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

var (
	emptyParseConf = func(in []byte) (conf interface{}, err error) {
		return string(in), nil
	}

	emptyFilter = func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		return
	}
)

func TestHTTPReqCall(t *testing.T) {
	InitConfCache(10 * time.Millisecond)
	SetRuleConfInTest(1, RuleConf{})

	builder := flatbuffers.NewBuilder(1024)
	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	b, err := HTTPReqCall(out, nil)
	assert.Nil(t, err)

	out = b.FinishedBytes()
	resp := hrc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(233), resp.Id())
	assert.Equal(t, hrc.ActionNONE, resp.ActionType())
}

func TestHTTPReqCall_FailedToParseConf(t *testing.T) {
	InitConfCache(1 * time.Millisecond)

	bazParseConf := func(in []byte) (conf interface{}, err error) {
		return nil, errors.New("ouch")
	}
	bazFilter := func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		w.Header().Add("foo", "bar")
		assert.Equal(t, "foo", conf.(string))
	}

	RegisterPlugin("baz", bazParseConf, bazFilter)

	builder := flatbuffers.NewBuilder(1024)
	bazName := builder.CreateString("baz")
	bazConf := builder.CreateString("")
	prepareConfWithData(builder, bazName, bazConf)

	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	b, err := HTTPReqCall(out, nil)
	assert.Nil(t, err)

	out = b.FinishedBytes()
	resp := hrc.GetRootAsResp(out, 0)
	assert.Equal(t, uint32(233), resp.Id())
	assert.Equal(t, hrc.ActionNONE, resp.ActionType())
}

func TestRegisterPlugin(t *testing.T) {
	type args struct {
		name string
		pc   ParseConfFunc
		sv   FilterFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "test_MissingParseConfMethod",
			args: args{
				name: "1",
				pc:   nil,
				sv:   emptyFilter,
			},
			wantErr: ErrMissingParseConfMethod,
		},
		{
			name: "test_MissingFilterMethod",
			args: args{
				name: "1",
				pc:   emptyParseConf,
				sv:   nil,
			},
			wantErr: ErrMissingFilterMethod,
		},
		{
			name: "test_MissingParseConfMethod&FilterMethod",
			args: args{
				name: "1",
				pc:   nil,
				sv:   nil,
			},
			wantErr: ErrMissingParseConfMethod,
		},
		{
			name: "test_MissingName&ParseConfMethod",
			args: args{
				name: "",
				pc:   nil,
				sv:   emptyFilter,
			},
			wantErr: ErrMissingName,
		},
		{
			name: "test_MissingName&FilterMethod",
			args: args{
				name: "",
				pc:   emptyParseConf,
				sv:   nil,
			},
			wantErr: ErrMissingName,
		},
		{
			name: "test_MissingAll",
			args: args{
				name: "",
				pc:   nil,
				sv:   nil,
			},
			wantErr: ErrMissingName,
		},
		{
			name: "test_plugin1",
			args: args{
				name: "plugin1",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: nil,
		},
		{
			name: "test_plugin1_again",
			args: args{
				name: "plugin1",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"plugin1"},
		},
		{
			name: "test_plugin2",
			args: args{
				name: "plugin2111%#@#",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: nil,
		},
		{
			name: "test_plugin3",
			args: args{
				name: "plugin311*%#@#",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: nil,
		},
		{
			name: "test_plugin3_again",
			args: args{
				name: "plugin311*%#@#",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"plugin311*%#@#"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RegisterPlugin(tt.args.name, tt.args.pc, tt.args.sv); !assert.Equal(t, tt.wantErr, err) {
				t.Errorf("RegisterPlugin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterPluginConcurrent(t *testing.T) {
	RegisterPlugin("test_concurrent-1", emptyParseConf, emptyFilter)
	RegisterPlugin("test_concurrent-2", emptyParseConf, emptyFilter)
	type args struct {
		name string
		pc   ParseConfFunc
		sv   FilterFunc
	}
	type test struct {
		name    string
		args    args
		wantErr error
	}
	tests := []test{
		{
			name: "test_concurrent-1",
			args: args{
				name: "test_concurrent-1",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"test_concurrent-1"},
		},
		{
			name: "test_concurrent-2#01",
			args: args{
				name: "test_concurrent-2",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"test_concurrent-2"},
		},
		{
			name: "test_concurrent-2#02",
			args: args{
				name: "test_concurrent-2",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"test_concurrent-2"},
		},
		{
			name: "test_concurrent-2#03",
			args: args{
				name: "test_concurrent-2",
				pc:   emptyParseConf,
				sv:   emptyFilter,
			},
			wantErr: ErrPluginRegistered{"test_concurrent-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 3; i++ {
				go func(tt test) {
					if err := RegisterPlugin(tt.args.name, tt.args.pc, tt.args.sv); !assert.Equal(t, tt.wantErr, err) {
						t.Errorf("RegisterPlugin() error = %v, wantErr %v", err, tt.wantErr)
					}
				}(tt)
			}

		})
	}
}

func TestFilter(t *testing.T) {
	InitConfCache(1 * time.Millisecond)

	fooParseConf := func(in []byte) (conf interface{}, err error) {
		return "foo", nil
	}
	fooFilter := func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		w.Header().Add("foo", "bar")
		assert.Equal(t, "foo", conf.(string))
	}
	barParseConf := func(in []byte) (conf interface{}, err error) {
		return "bar", nil
	}
	barFilter := func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request) {
		r.Header().Set("foo", "bar")
		assert.Equal(t, "bar", conf.(string))
	}

	RegisterPlugin("foo", fooParseConf, fooFilter)
	RegisterPlugin("bar", barParseConf, barFilter)

	builder := flatbuffers.NewBuilder(1024)
	fooName := builder.CreateString("foo")
	fooConf := builder.CreateString("foo")
	barName := builder.CreateString("bar")
	barConf := builder.CreateString("bar")
	prepareConfWithData(builder, fooName, fooConf, barName, barConf)

	res, _ := GetRuleConf(1)
	hrc.ReqStart(builder)
	hrc.ReqAddId(builder, 233)
	hrc.ReqAddConfToken(builder, 1)
	r := hrc.ReqEnd(builder)
	builder.Finish(r)
	out := builder.FinishedBytes()

	req := inHTTP.CreateRequest(out)
	resp := inHTTP.CreateResponse()
	filter(res, resp, req)

	assert.Equal(t, "bar", resp.Header().Get("foo"))
	assert.Equal(t, "", req.Header().Get("foo"))

	req = inHTTP.CreateRequest(out)
	resp = inHTTP.CreateResponse()
	prepareConfWithData(builder, barName, barConf, fooName, fooConf)
	res, _ = GetRuleConf(2)
	filter(res, resp, req)

	assert.Equal(t, "bar", resp.Header().Get("foo"))
	assert.Equal(t, "bar", req.Header().Get("foo"))
}
