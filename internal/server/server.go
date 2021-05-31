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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/apache/apisix-go-plugin-runner/internal/log"
	"github.com/apache/apisix-go-plugin-runner/internal/plugin"
	"github.com/apache/apisix-go-plugin-runner/internal/util"
	flatbuffers "github.com/google/flatbuffers/go"
)

const (
	HeaderLen   = 4
	MaxDataSize = 2<<24 - 1

	SockAddrEnv     = "APISIX_LISTEN_ADDRESS"
	ConfCacheTTLEnv = "APISIX_CONF_EXPIRE_TIME"
)

const (
	RPCError = iota
	RPCPrepareConf
	RPCHTTPReqCall
)

func readErr(n int, err error, required int) bool {
	if n < required {
		err = errors.New("truncated")
	}
	if err != nil && err != io.EOF {
		log.Errorf("read: %s", err)
		return true
	}
	return false
}

func writeErr(n int, err error) {
	if err != nil {
		log.Errorf("write: %s", err)
	}
}

func handleConn(c net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic recovered: %s", err)
		}
	}()

	log.Infof("Client connected (%s)", c.RemoteAddr().Network())
	defer c.Close()

	header := make([]byte, HeaderLen)
	for {
		n, err := c.Read(header)
		if readErr(n, err, HeaderLen) {
			break
		}

		ty := header[0]
		// we only use last 3 bytes to store the length, so the first byte is
		// consider zero
		header[0] = 0
		length := binary.BigEndian.Uint32(header)

		log.Infof("receive rpc type: %d data length: %d", ty, length)

		buf := make([]byte, length)
		n, err = c.Read(buf)
		if readErr(n, err, int(length)) {
			break
		}

		var bd *flatbuffers.Builder
		switch ty {
		case RPCPrepareConf:
			bd, err = plugin.PrepareConf(buf)
		case RPCHTTPReqCall:
			bd, err = plugin.HTTPReqCall(buf)
		default:
			err = UnknownType{ty}
		}

		out := bd.FinishedBytes()
		size := len(out)
		if size > MaxDataSize {
			err = fmt.Errorf("the max length of data is %d but got %d", MaxDataSize, size)
		}

		if err != nil {
			log.Errorf("%s", err)

			ty = RPCError
			util.PutBuilder(bd)
			bd = ReportError(err)
			out = bd.FinishedBytes()
		}

		binary.BigEndian.PutUint32(header, uint32(size))
		header[0] = ty

		n, err = c.Write(header)
		if err != nil {
			writeErr(n, err)
			util.PutBuilder(bd)
			break
		}

		n, err = c.Write(out)
		if err != nil {
			writeErr(n, err)
			util.PutBuilder(bd)
			break
		}
		util.PutBuilder(bd)
	}
}

func getConfCacheTTL() time.Duration {
	ttl := os.Getenv(ConfCacheTTLEnv)
	n, err := strconv.Atoi(ttl)
	if err != nil || n <= 0 {
		log.Errorf("invalid cache ttl: %s", ttl)
		return 0
	}
	return time.Duration(n) * time.Second
}

func getSockAddr() string {
	path := os.Getenv(SockAddrEnv)
	if !strings.HasPrefix(path, "unix:") {
		log.Errorf("invalid socket address: %s", path)
		return ""
	}
	return path[len("unix:"):]
}

func Run() {
	ttl := getConfCacheTTL()
	if ttl == 0 {
		log.Fatalf("A valid conf cache ttl should be set via environment variable %s",
			ConfCacheTTLEnv)
	}
	log.Infof("conf cache ttl is %v", ttl)

	plugin.InitConfCache(ttl)

	sockAddr := getSockAddr()
	if sockAddr == "" {
		log.Fatalf("A valid socket address should be set via environment variable %s", SockAddrEnv)
	}
	log.Infof("listening to %s", sockAddr)

	// clean up sock file created by others
	if err := os.RemoveAll(sockAddr); err != nil {
		log.Fatalf("remove file %s: %s", sockAddr, err)
	}
	// clean up sock file created by me
	defer func() {
		if err := os.RemoveAll(sockAddr); err != nil {
			log.Errorf("remove file %s: %s", sockAddr, err)
		}
	}()

	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		log.Fatalf("listen %s: %s", sockAddr, err)
	}
	defer l.Close()

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			conn, err := l.Accept()
			if err != nil {
				log.Errorf("accept: %s", err)
				continue
			}

			go handleConn(conn)
		}
	}()

	sig := <-quit
	log.Infof("server receive %s and exit", sig.String())
	close(done)
}
