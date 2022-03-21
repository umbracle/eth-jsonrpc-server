package jsonrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNet_Register(t *testing.T) {
	net := NewNet(nil)
	d := &Dispatcher{logger: defaultNullLogger}
	d.Register("net", net)
}

func TestNet_Version(t *testing.T) {
	net := NewNet(&mockBackend{chainID: 1})

	version, err := net.Version()
	assert.NoError(t, err)

	assert.Equal(t, version.Uint64(), uint64(1))
}

func TestNet_Listening(t *testing.T) {
	net := NewNet(&mockBackend{listening: true})

	listening, err := net.Listening()
	assert.NoError(t, err)

	assert.Equal(t, listening, true)
}

func TestNet_PeerCount(t *testing.T) {
	net := NewNet(&mockBackend{peerCount: 10})

	peerCount, err := net.PeerCount()
	assert.NoError(t, err)

	assert.Equal(t, peerCount.Uint64(), uint64(10))
}

type mockBackend struct {
	chainID   uint64
	peerCount int
	listening bool
}

func (m *mockBackend) ChainID() uint64 {
	return m.chainID
}

func (m *mockBackend) PeerCount() int {
	return m.peerCount
}

func (m *mockBackend) Listening() bool {
	return m.listening
}
