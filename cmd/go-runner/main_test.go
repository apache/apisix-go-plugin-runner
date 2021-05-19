package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	args := []string{"version", "--long"}
	os.Args = append([]string{"cmd"}, args...)

	var b bytes.Buffer
	InfoOut = &b
	main()

	assert.True(t, strings.Contains(b.String(), "Building OS/Arch"))
}
