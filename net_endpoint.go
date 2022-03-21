package jsonrpc

// NetBackend is the backend for the Net namespace
type NetBackend interface {
	ChainID() uint64
	PeerCount() int
	Listening() bool
}

// Net is the net jsonrpc endpoint
type Net struct {
	b NetBackend
}

func NewNet(b NetBackend) *Net {
	return &Net{b: b}
}

// Version returns the current network id
func (n *Net) Version() (*argUint64, error) {
	return argUintPtr(n.b.ChainID()), nil
}

// Listening returns true if client is actively listening for network connections
func (n *Net) Listening() (bool, error) {
	return n.b.Listening(), nil
}

// PeerCount returns number of peers currently connected to the client
func (n *Net) PeerCount() (*argUint64, error) {
	return argUintPtr(uint64(n.b.PeerCount())), nil
}
