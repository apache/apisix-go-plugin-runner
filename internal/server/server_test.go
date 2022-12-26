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

package server

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-go-plugin-runner/internal/plugin"
	"github.com/apache/apisix-go-plugin-runner/internal/util"
)

func TestGetSockAddr(t *testing.T) {
	os.Unsetenv(SockAddrEnv)
	assert.Equal(t, "", getSockAddr())

	os.Setenv(SockAddrEnv, "unix:/tmp/x.sock")
	assert.Equal(t, "/tmp/x.sock", getSockAddr())
}

func TestGetConfCacheTTL(t *testing.T) {
	os.Unsetenv(ConfCacheTTLEnv)
	assert.Equal(t, 4320*time.Second, getConfCacheTTL())

	os.Setenv(ConfCacheTTLEnv, "12")
	assert.Equal(t, 14*time.Second, getConfCacheTTL())

	os.Setenv(ConfCacheTTLEnv, "1a")
	assert.Equal(t, time.Duration(0), getConfCacheTTL())
}

func TestDispatchRPC_UnknownType(t *testing.T) {
	bd, ty := dispatchRPC(126, []byte(""), nil)
	err := UnknownType{126}
	expectBd := ReportError(err)
	assert.Equal(t, expectBd.FinishedBytes(), bd.FinishedBytes())
	assert.Equal(t, ty, byte(util.RPCError))
}

func TestDispatchRPC_KnownType(t *testing.T) {
	bd := util.GetBuilder()
	hrc.ReqStart(bd)
	hrc.ReqAddConfToken(bd, 1)
	r := hrc.ReqEnd(bd)
	bd.Finish(r)

	_, ty := dispatchRPC(util.RPCHTTPReqCall, bd.FinishedBytes(), nil)
	assert.Equal(t, ty, byte(util.RPCError))
}

func TestDispatchRPC_OutTooLarge(t *testing.T) {
	builder := util.GetBuilder()
	bodyVec := builder.CreateByteVector(make([]byte, util.MaxDataSize+1))
	hrc.StopStart(builder)
	hrc.StopAddBody(builder, bodyVec)
	stop := hrc.StopEnd(builder)

	hrc.RespStart(builder)
	hrc.RespAddId(builder, 1)
	hrc.RespAddActionType(builder, hrc.ActionStop)
	hrc.RespAddAction(builder, stop)
	res := hrc.RespEnd(builder)
	builder.Finish(res)
}

func TestRun(t *testing.T) {
	path := "/tmp/x.sock"
	addr := "unix:" + path
	os.Setenv(SockAddrEnv, addr)
	os.Setenv(ConfCacheTTLEnv, "60")

	go func() {
		Run()
	}()

	time.Sleep(100 * time.Millisecond)

	stat, err := os.Stat(path)
	assert.True(t, stat.Mode().Perm() == 0766)

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
		// header with bad body
		{append(header, bytes.Repeat([]byte{1, 2}, 16)...)},
	}

	for _, c := range cases {
		conn, err := net.DialTimeout("unix", addr[len("unix:"):], 1*time.Second)
		assert.NotNil(t, conn, err)
		defer conn.Close()
		util.WriteBytes(conn, c.header, len(c.header))
	}

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	time.Sleep(10 * time.Millisecond)

	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func init() {
	plugin.InitConfCache(time.Second)
}
