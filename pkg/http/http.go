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

package http

import (
	"context"
	"net"
	"net/http"
	"net/url"
)

// Request represents the HTTP request received by APISIX.
// We don't use net/http's Request because it doesn't suit our purpose.
// Take `Request.Header` as an example:
//
// 1. We need to record any change to the request headers. As the Request.Header
// is not an interface, there is not way to inject our special tracker.
//
// 2. As the author of fasthttp pointed out, "headers are stored in a map[string][]string.
// So the server must parse all the headers, ...". The official API is suboptimal, which
// is even worse in our case as it is not a real HTTP server.
type Request interface {
	// ID returns the request id
	ID() uint32

	// SrcIP returns the client's IP
	SrcIP() net.IP
	// Method returns the HTTP method (GET, POST, PUT, etc.)
	Method() string
	// Path returns the path part of the client's URI (without query string and the other parts)
	// It won't be equal to the one in the Request-Line sent by the client if it has
	// been rewritten by APISIX
	Path() []byte
	// SetPath is the setter for Path
	SetPath([]byte)
	// Header returns the HTTP headers
	Header() Header
	// Args returns the query string
	Args() url.Values

	// Var returns the value of a Nginx variable, like `r.Var("request_time")`
	//
	// To fetch the value, the runner will look up the request's cache first. If not found,
	// the runner will ask it from the APISIX. If the RPC call is failed, an error in
	// pkg/common.ErrConnClosed type is returned.
	Var(name string) ([]byte, error)

	// Body returns HTTP request body
	//
	// To fetch the value, the runner will look up the request's cache first. If not found,
	// the runner will ask it from the APISIX. If the RPC call is failed, an error in
	// pkg/common.ErrConnClosed type is returned.
	Body() ([]byte, error)

	// Context returns the request's context.
	//
	// The returned context is always non-nil; it defaults to the
	// background context.
	//
	// For run plugin, the context controls cancellation.
	Context() context.Context
	// RespHeader returns an http.Header which allows you to add or set response headers before reaching the upstream.
	// Some built-in headers would not take effect, like `connection`,`content-length`,`transfer-encoding`,`location,server`,`www-authenticate`,`content-encoding`,`content-type`,`content-location` and `content-language`
	RespHeader() http.Header
}

// Response represents the HTTP response from the upstream received by APISIX.
// In order to avoid semantic misunderstanding,
// we also use Response to represent the rewritten response from Plugin Runner.
// Therefore, any instance that implements the Response interface will be readable and rewritable.
type Response interface {
	// ID returns the request id
	ID() uint32

	// StatusCode returns the response code
	StatusCode() int

	// Header returns the response header.
	//
	// It allows you to add or set response headers before reaching the client.
	Header() Header

	// Var returns the value of a Nginx variable, like `r.Var("request_time")`
	//
	// To fetch the value, the runner will look up the request's cache first. If not found,
	// the runner will ask it from the APISIX. If the RPC call is failed, an error in
	// pkg/common.ErrConnClosed type is returned.
	Var(name string) ([]byte, error)

	// ReadBody returns origin HTTP response body
	//
	// To fetch the value, the runner will look up the request's cache first. If not found,
	// the runner will ask it from the APISIX. If the RPC call is failed, an error in
	// pkg/common.ErrConnClosed type is returned.
	//
	// It was not named `Body`
	// because `Body` was already occupied in earlier interface implementations.
	ReadBody() ([]byte, error)

	// Write rewrites the origin response data.
	//
	// Unlike `ResponseWriter.Write`, we don't need to WriteHeader(http.StatusOK)
	// before writing the data
	// Because APISIX will convert code 0 to 200.
	Write(b []byte) (int, error)

	// WriteHeader rewrites the origin response StatusCode
	//
	// WriteHeader can't override written status.
	WriteHeader(statusCode int)
}

// Header is like http.Header, but only implements the subset of its methods
type Header interface {
	// Set sets the header entries associated with key to the single element value.
	// It replaces any existing values associated with key.
	// The key is case insensitive
	Set(key, value string)
	// Del deletes the values associated with key. The key is case insensitive
	Del(key string)
	// Get gets the first value associated with the given key.
	// If there are no values associated with the key, Get returns "".
	// It is case insensitive
	Get(key string) string
	// View returns the internal structure. It is expected for read operations. Any write operation
	// won't be recorded
	View() http.Header

	// TODO: support Add
}
