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
	"fmt"
	"net"
	"net/http"
	"sync"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"

	inHTTP "github.com/apache/apisix-go-plugin-runner/internal/http"
	"github.com/apache/apisix-go-plugin-runner/internal/util"
	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

type ParseConfFunc func(in []byte) (conf interface{}, err error)
type RequestFilterFunc func(conf interface{}, w http.ResponseWriter, r pkgHTTP.Request)
type ResponseFilterFunc func(conf interface{}, w pkgHTTP.Response)

type pluginOpts struct {
	ParseConf      ParseConfFunc
	RequestFilter  RequestFilterFunc
	ResponseFilter ResponseFilterFunc
}

type pluginRegistries struct {
	sync.Mutex
	opts map[string]*pluginOpts
}

type ErrPluginRegistered struct {
	name string
}

func (err ErrPluginRegistered) Error() string {
	return fmt.Sprintf("plugin %s registered", err.name)
}

var (
	pluginRegistry = pluginRegistries{opts: map[string]*pluginOpts{}}

	ErrMissingName                 = errors.New("missing name")
	ErrMissingParseConfMethod      = errors.New("missing ParseConf method")
	ErrMissingRequestFilterMethod  = errors.New("missing RequestFilter method")
	ErrMissingResponseFilterMethod = errors.New("missing ResponseFilter method")
)

func RegisterPlugin(name string, pc ParseConfFunc, sv RequestFilterFunc, rsv ResponseFilterFunc) error {
	log.Infof("register plugin %s", name)

	if name == "" {
		return ErrMissingName
	}
	if pc == nil {
		return ErrMissingParseConfMethod
	}
	if sv == nil {
		return ErrMissingRequestFilterMethod
	}
	if rsv == nil {
		return ErrMissingResponseFilterMethod
	}

	opt := &pluginOpts{
		ParseConf:      pc,
		RequestFilter:  sv,
		ResponseFilter: rsv,
	}
	pluginRegistry.Lock()
	defer pluginRegistry.Unlock()
	if _, found := pluginRegistry.opts[name]; found {
		return ErrPluginRegistered{name}
	}
	pluginRegistry.opts[name] = opt
	return nil
}

func findPlugin(name string) *pluginOpts {
	if opt, found := pluginRegistry.opts[name]; found {
		return opt
	}
	return nil
}

func filter(conf RuleConf, w *inHTTP.ReqResponse, r pkgHTTP.Request) error {
	for _, c := range conf {
		plugin := findPlugin(c.Name)
		if plugin == nil {
			log.Warnf("can't find plugin %s, skip", c.Name)
			continue
		}

		log.Infof("run plugin %s", c.Name)

		plugin.RequestFilter(c.Value, w, r)

		if w.HasChange() {
			// response is generated, no need to continue
			break
		}
	}
	return nil
}

func reportAction(id uint32, req *inHTTP.Request, resp *inHTTP.ReqResponse) *flatbuffers.Builder {
	builder := util.GetBuilder()

	if resp != nil && resp.FetchChanges(id, builder) {
		return builder
	}

	if req != nil && req.FetchChanges(id, builder) {
		return builder
	}

	hrc.RespStart(builder)
	hrc.RespAddId(builder, id)
	res := hrc.RespEnd(builder)
	builder.Finish(res)
	return builder
}

func HTTPReqCall(buf []byte, conn net.Conn) (*flatbuffers.Builder, error) {
	req := inHTTP.CreateRequest(buf)
	req.BindConn(conn)
	defer inHTTP.ReuseRequest(req)

	resp := inHTTP.CreateReqResponse()
	defer inHTTP.ReuseReqResponse(resp)

	token := req.ConfToken()
	conf, err := GetRuleConf(token)
	if err != nil {
		return nil, err
	}

	err = filter(conf, resp, req)
	if err != nil {
		return nil, err
	}

	id := req.ID()
	builder := reportAction(id, req, resp)
	return builder, nil
}
