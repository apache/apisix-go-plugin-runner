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
package http

// Request represents the HTTP request received by APISIX.
// We don't use net/http's Request because it doesn't suit our purpose.
// Take `Request.Header` as an example:
//
// 1. We need to record any change to the request headers. As the Request.Header
// is not an interface, there is not way to inject our special tracker.
// 2. As the author of fasthttp pointed out, "headers are stored in a map[string][]string.
// So the server must parse all the headers, ...". The official API is suboptimal, which
// is even worse in our case as it is not a real HTTP server.
type Request interface {
	// Id returns the request id
	Id() uint32
}
