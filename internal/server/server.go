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
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/internal/plugin"
	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

const (
	SockAddrEnv     = "APISIX_LISTEN_ADDRESS"
	ConfCacheTTLEnv = "APISIX_CONF_EXPIRE_TIME"
)

type handler func(buf []byte, conn net.Conn) (*flatbuffers.Builder, error)

var (
	typeHandlerMap = map[byte]handler{
		util.RPCPrepareConf: func(buf []byte, conn net.Conn) (*flatbuffers.Builder, error) {
			return plugin.PrepareConf(buf)
		},
		util.RPCHTTPReqCall: func(buf []byte, conn net.Conn) (*flatbuffers.Builder, error) {
			return plugin.HTTPReqCall(buf, conn)
		},
	}
)

func generateErrorReport(err error) *flatbuffers.Builder {
	if err == ttlcache.ErrNotFound {
		log.Warnf("%s", err)
	} else {
		log.Errorf("%s", err)
	}

	return ReportError(err)
}

func recoverPanic() {
	if err := recover(); err != nil {
		log.Errorf("panic recovered: %s", err)
	}
}

func dispatchRPC(ty byte, in []byte, conn net.Conn) *flatbuffers.Builder {
	var err error
	var bd *flatbuffers.Builder
	hl, ok := typeHandlerMap[ty]
	if !ok {
		err = UnknownType{ty}
	} else {
		bd, err = hl(in, conn)
	}

	if err != nil {
		bd = generateErrorReport(err)
	} else {
		bd = checkIfDataTooLarge(bd)
	}
	return bd
}

func checkIfDataTooLarge(bd *flatbuffers.Builder) *flatbuffers.Builder {
	out := bd.FinishedBytes()
	size := len(out)
	if size > util.MaxDataSize {
		err := fmt.Errorf("the max length of data is %d but got %d", util.MaxDataSize, size)
		util.PutBuilder(bd)
		bd = generateErrorReport(err)
	}
	return bd
}

func handleConn(c net.Conn) {
	defer recoverPanic()

	log.Infof("Client connected (%s)", c.RemoteAddr().Network())
	defer c.Close()

	header := make([]byte, util.HeaderLen)
	for {
		n, err := c.Read(header)
		if util.ReadErr(n, err, util.HeaderLen) {
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
		if util.ReadErr(n, err, int(length)) {
			break
		}

		bd := dispatchRPC(ty, buf, c)
		out := bd.FinishedBytes()
		size := len(out)
		binary.BigEndian.PutUint32(header, uint32(size))
		header[0] = ty

		n, err = c.Write(header)
		if err != nil {
			util.WriteErr(n, err)
			break
		}

		n, err = c.Write(out)
		if err != nil {
			util.WriteErr(n, err)
			break
		}

		util.PutBuilder(bd)
	}
}

func getConfCacheTTL() time.Duration {
	// ensure the conf cached in the runner expires after the token in APISIX
	amplificationFactor := 1.2
	ttl := os.Getenv(ConfCacheTTLEnv)
	if ttl == "" {
		return time.Duration(3600*amplificationFactor) * time.Second
	}

	n, err := strconv.Atoi(ttl)
	if err != nil || n <= 0 {
		log.Errorf("invalid cache ttl: %s", ttl)
		return 0
	}
	return time.Duration(float64(n)*amplificationFactor) * time.Second
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
	log.Warnf("conf cache ttl is %v", ttl)

	plugin.InitConfCache(ttl)

	sockAddr := getSockAddr()
	if sockAddr == "" {
		log.Fatalf("A valid socket address should be set via environment variable %s", SockAddrEnv)
	}
	log.Warnf("listening to %s", sockAddr)

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

	// the default socket permission is 0755, which prevents the 'nobody' worker process
	// from writing to it if the APISIX is run under root.
	err = os.Chmod(sockAddr, 0766)
	if err != nil {
		log.Fatalf("can't change mod for file %s: %s", sockAddr, err)
	}

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			conn, err := l.Accept()

			select {
			case <-done:
				// don't report the "use of closed network connection" error when the server
				// is exiting.
				return
			default:
			}

			if err != nil {
				log.Errorf("accept: %s", err)
				continue
			}

			go handleConn(conn)
		}
	}()

	sig := <-quit
	log.Warnf("server receive %s and exit", sig.String())
	close(done)
}
