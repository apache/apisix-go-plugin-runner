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

package httptest

import (
	"bytes"
	"net/http"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
)

// ResponseRecorder is an implementation of pkgHTTP.Response that
// records its mutations for later inspection in tests.
type ResponseRecorder struct {
	// Code is the HTTP response code set at initialization.
	Code int

	// HeaderMap contains the headers explicitly set by the Handler.
	// It is an internal detail.
	HeaderMap pkgHTTP.Header

	// Body is the buffer to which the Handler's Write calls are sent.
	// If nil, the Writes are silently discarded.
	Body *bytes.Buffer

	// OriginBody is the response body received by APISIX from upstream.
	OriginBody []byte

	Vars map[string][]byte

	statusCode int
	id         uint32
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap: newHeader(),
		Body:      new(bytes.Buffer),
	}
}

// ID is APISIX rpc's id.
func (rw *ResponseRecorder) ID() uint32 {
	return rw.id
}

// StatusCode returns the response code.
//
// Note that if a Handler never calls WriteHeader,
// this will be initial status code, rather than the implicit
// http.StatusOK.
func (rw *ResponseRecorder) StatusCode() int {
	if rw.statusCode == 0 {
		return rw.Code
	}

	return rw.statusCode
}

// Header implements pkgHTTP.Response. It returns the response
// headers to mutate within a handler.
func (rw *ResponseRecorder) Header() pkgHTTP.Header {
	m := rw.HeaderMap
	if m == nil {
		rw.HeaderMap = newHeader()
	}
	return m
}

// Write implements pkgHTTP.Response.
// The data in buf is written to rw.Body, if not nil.
func (rw *ResponseRecorder) Write(buf []byte) (int, error) {
	if rw.Body == nil {
		rw.Body = &bytes.Buffer{}
	}
	return rw.Body.Write(buf)
}

// Var implements pkgHTTP.Response.
func (rw *ResponseRecorder) Var(key string) ([]byte, error) {
	if rw.Vars == nil {
		rw.Vars = make(map[string][]byte)
	}
	return rw.Vars[key], nil
}

// ReadBody implements pkgHTTP.Response.
func (rw *ResponseRecorder) ReadBody() ([]byte, error) {
	if rw.OriginBody == nil {
		rw.OriginBody = make([]byte, 0)
	}
	return rw.OriginBody, nil
}

// WriteHeader implements pkgHTTP.Response.
// The statusCode is only allowed to be written once.
func (rw *ResponseRecorder) WriteHeader(code int) {
	if rw.statusCode != 0 {
		return
	}

	rw.statusCode = code
}

type Header struct {
	http.Header
}

func (h *Header) View() http.Header {
	return h.Header
}

func newHeader() *Header {
	return &Header{
		Header: http.Header{},
	}
}
