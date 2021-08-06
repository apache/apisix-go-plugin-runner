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
package plugins

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimitReq(t *testing.T) {
	in := []byte(`{"rate":5,"burst":1}`)
	lr := &LimitReq{}
	conf, err := lr.ParseConf(in)
	assert.Nil(t, err)

	start := time.Now()
	n := 6
	var wg sync.WaitGroup
	res := make([]*http.Response, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			w := httptest.NewRecorder()
			lr.Filter(conf, w, nil)
			resp := w.Result()
			res[i] = resp
			wg.Done()
		}(i)
	}
	wg.Wait()

	rejectN := 0
	for _, r := range res {
		if r.StatusCode == 503 {
			rejectN++
		}
	}
	assert.Equal(t, 0, rejectN)
	t.Logf("Start: %v, now: %v", start, time.Now())
	assert.True(t, time.Now().Sub(start) >= 1*time.Second)
}

func TestLimitReq_YouShouldNotPass(t *testing.T) {
	in := []byte(`{}`)
	lr := &LimitReq{}
	conf, err := lr.ParseConf(in)
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	lr.Filter(conf, w, nil)
	resp := w.Result()
	assert.Equal(t, 503, resp.StatusCode)
}
