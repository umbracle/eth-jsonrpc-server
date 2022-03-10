package jsonrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPServer(t *testing.T) {
	config := &Config{
		Addr: defaultHttpAddr,
	}
	_, err := NewJSONRPC(defaultNullLogger, config, nil)
	assert.NoError(t, err)
}
