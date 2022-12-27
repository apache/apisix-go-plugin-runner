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

package util

import (
	"fmt"
	"io"
	"net"

	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

const (
	HeaderLen   = 4
	MaxDataSize = 2<<24 - 1
)

const (
	RPCError = iota
	RPCPrepareConf
	RPCHTTPReqCall
	RPCExtraInfo
	RPCHTTPRespCall
)

type RPCResult struct {
	Err     error
	Builder *flatbuffers.Builder
}

// Use struct if the result is not only []byte
type ExtraInfoResult []byte

func ReadErr(n int, err error, required int) bool {
	if 0 < n && n < required {
		err = fmt.Errorf("truncated, only get the first %d bytes", n)
	}
	if err != nil {
		if err != io.EOF {
			log.Errorf("read: %s", err)
		}
		return true
	}
	return false
}

func WriteErr(n int, err error) {
	if err != nil {
		log.Errorf("write: %s", err)
	}
}

func ReadBytes(c net.Conn, b []byte, n int) (int, error) {
	l := 0
	for l < n {
		tmp, err := c.Read(b[l:])
		if err != nil {
			return l + tmp, err
		}
		l += tmp
	}
	return l, nil
}

func WriteBytes(c net.Conn, b []byte, n int) (int, error) {
	l := 0
	for l < n {
		tmp, err := c.Write(b[l:])
		if err != nil {
			return l + tmp, err
		}
		l += tmp
	}
	return l, nil
}
