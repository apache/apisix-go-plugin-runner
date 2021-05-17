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
package server

import (
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSockAddr(t *testing.T) {
	os.Unsetenv(SockAddrEnv)
	assert.Equal(t, "", getSockAddr())

	os.Setenv(SockAddrEnv, "/tmp/x.sock")
	assert.Equal(t, "/tmp/x.sock", getSockAddr())
}

func TestRun(t *testing.T) {
	addr := "/tmp/x.sock"
	os.Setenv(SockAddrEnv, addr)

	go func() {
		Run()
	}()

	time.Sleep(100 * time.Millisecond)
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(32))
	header[0] = 1
	cases := []struct {
		header []byte
	}{
		// dad header
		{[]byte("a")},
		// header without body
		{header},
		// header without body truncated
		{append(header, 32)},
	}

	for _, c := range cases {
		conn, err := net.DialTimeout("unix", addr, 1*time.Second)
		assert.NotNil(t, conn, err)
		conn.Write(c.header)
	}
}
