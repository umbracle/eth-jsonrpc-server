package jsonrpc

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeb3_Register(t *testing.T) {
	web3 := NewWeb3(nil)
	d := &Dispatcher{logger: defaultNullLogger}
	d.Register("web3", web3)
}

func TestWeb3_Sha3(t *testing.T) {
	web3 := NewWeb3(nil)

	expect, err := hex.DecodeString("22ae6da6b482f9b1b19b0b897c3fd43884180a1c5ee361e1107a1bc635649dda")
	assert.NoError(t, err)

	found, err := web3.Sha3(argBytes([]byte{0x1, 0x2}))
	assert.NoError(t, err)

	assert.Equal(t, expect, found.Bytes())
}

func TestWeb3_ClientVersion(t *testing.T) {
	b := &mockWeb3Version{version: "client-version"}
	web3 := NewWeb3(b)

	expect, err := web3.ClientVersion()
	assert.NoError(t, err)

	assert.Equal(t, expect, "client-version")
}

type mockWeb3Version struct {
	version string
}

func (m *mockWeb3Version) Version() string {
	return m.version
}
