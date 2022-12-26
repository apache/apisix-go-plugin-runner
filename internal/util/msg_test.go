package util

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadAndWriteBytes(t *testing.T) {
	path := "/tmp/test.sock"
	server, err := net.Listen("unix", path)
	assert.NoError(t, err)
	defer server.Close()

	// transfer large enough data
	n := 10000000

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	in := make([]byte, n)
	for i := range in {
		in[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	go func() {
		client, err := net.DialTimeout("unix", path, 1*time.Second)
		assert.NoError(t, err)
		defer client.Close()
		WriteBytes(client, in, len(in))
	}()

	fd, err := server.Accept()
	assert.NoError(t, err)
	out := make([]byte, n)
	ReadBytes(fd, out, n)
	assert.Equal(t, in, out)
}
