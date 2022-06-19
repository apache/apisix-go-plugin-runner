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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFaultInjection(t *testing.T) {
	in := []byte(`{"http_status":400, "body":"hello"}`)
	fi := &FaultInjection{}
	conf, err := fi.ParseConf(in)
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	fi.RequestFilter(conf, w, nil)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "hello", string(body))
}

func TestFaultInjection_Percentage(t *testing.T) {
	in := []byte(`{"http_status":400, "percentage":0}`)
	fi := &FaultInjection{}
	conf, err := fi.ParseConf(in)
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	fi.RequestFilter(conf, w, nil)
	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)
}
