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

package plugins

import (
	"io/ioutil"
	"testing"

	pkgHTTPTest "github.com/apache/apisix-go-plugin-runner/pkg/httptest"

	"github.com/stretchr/testify/assert"
)

func TestResponseRewrite(t *testing.T) {
	in := []byte(`{"status":200, "headers":{"X-Server-Id":"9527"},"body":"response rewrite"}`)
	rr := &ResponseRewrite{}
	conf, err := rr.ParseConf(in)
	assert.Nil(t, err)
	assert.Equal(t, 200, conf.(ResponseRewriteConf).Status)
	assert.Equal(t, "9527", conf.(ResponseRewriteConf).Headers["X-Server-Id"])
	assert.Equal(t, "response rewrite", conf.(ResponseRewriteConf).Body)

	w := pkgHTTPTest.NewRecorder()
	w.Code = 502
	w.HeaderMap.Set("X-Resp-A6-Runner", "Java")
	rr.ResponseFilter(conf, w)

	body, _ := ioutil.ReadAll(w.Body)
	assert.Equal(t, 200, w.StatusCode())
	assert.Equal(t, "Go", w.Header().Get("X-Resp-A6-Runner"))
	assert.Equal(t, "9527", w.Header().Get("X-Server-Id"))
	assert.Equal(t, "response rewrite", string(body))
}

func TestResponseRewrite_BadConf(t *testing.T) {
	in := []byte(``)
	rr := &ResponseRewrite{}
	_, err := rr.ParseConf(in)
	assert.NotNil(t, err)
}

func TestResponseRewrite_ConfEmpty(t *testing.T) {
	in := []byte(`{}`)
	rr := &ResponseRewrite{}
	conf, err := rr.ParseConf(in)
	assert.Nil(t, err)
	assert.Equal(t, 0, conf.(ResponseRewriteConf).Status)
	assert.Equal(t, 0, len(conf.(ResponseRewriteConf).Headers))
	assert.Equal(t, "", conf.(ResponseRewriteConf).Body)

	w := pkgHTTPTest.NewRecorder()
	w.Code = 502
	w.HeaderMap.Set("X-Resp-A6-Runner", "Java")
	rr.ResponseFilter(conf, w)
	assert.Equal(t, 502, w.StatusCode())
	assert.Equal(t, "Go", w.Header().Get("X-Resp-A6-Runner"))
	assert.Equal(t, "", conf.(ResponseRewriteConf).Body)
}

func TestResponseRewrite_ReplaceGlobal(t *testing.T) {
	in := []byte(`{"filters":[{"regex":"world","scope":"global","replace":"golang"}]}`)
	rr := &ResponseRewrite{}
	conf, err := rr.ParseConf(in)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(conf.(ResponseRewriteConf).Filters))

	w := pkgHTTPTest.NewRecorder()
	w.Code = 200
	w.OriginBody = []byte("hello world world")
	rr.ResponseFilter(conf, w)
	assert.Equal(t, 200, w.StatusCode())
	body, _ := ioutil.ReadAll(w.Body)
	assert.Equal(t, "hello golang golang", string(body))
}

func TestResponseRewrite_ReplaceOnce(t *testing.T) {
	in := []byte(`{"filters":[{"regex":"world","scope":"once","replace":"golang"}]}`)
	rr := &ResponseRewrite{}
	conf, err := rr.ParseConf(in)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(conf.(ResponseRewriteConf).Filters))

	w := pkgHTTPTest.NewRecorder()
	w.Code = 200
	w.OriginBody = []byte("hello world world")
	rr.ResponseFilter(conf, w)
	assert.Equal(t, 200, w.StatusCode())
	body, _ := ioutil.ReadAll(w.Body)
	assert.Equal(t, "hello golang world", string(body))
}
