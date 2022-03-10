package jsonrpc

import (
	"golang.org/x/crypto/sha3"
)

// Web3Backend is the backend for the Net namespace
type Web3Backend interface {
	Version() string
}

// Web3 is the web3 jsonrpc endpoint
type Web3 struct {
	b Web3Backend
}

func NewWeb3(b Web3Backend) *Web3 {
	return &Web3{b: b}
}

// ClientVersion returns the version of the web3 client (web3_clientVersion)
func (w *Web3) ClientVersion() (string, error) {
	return w.b.Version(), nil
}

// Sha3 returns Keccak-256 (not the standardized SHA3-256) of the given data
func (w *Web3) Sha3(val argBytes) (*argBytes, error) {
	h := sha3.NewLegacyKeccak256()
	h.Write(val)
	res := h.Sum(nil)

	return argBytesPtr(res), nil
}
