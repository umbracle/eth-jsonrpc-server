package jsonrpc

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/ethgo"
)

func TestFilter_Log(t *testing.T) {
	store := newMockStore()

	m := NewFilterManager(nil, store)
	go m.Run()

	id := m.addFilter(&LogFilter{
		Topics: [][]ethgo.Hash{
			{hash1},
		},
	}, nil)

	store.emitEvent(&mockEvent{
		NewChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: hash1,
				},
				receipts: []*ethgo.Receipt{
					{
						Logs: []*ethgo.Log{
							{
								Topics: []ethgo.Hash{
									hash1,
								},
							},
						},
					},
				},
			},
		},
		OldChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: hash2,
				},
				receipts: []*ethgo.Receipt{
					{
						Logs: []*ethgo.Log{
							{
								Topics: []ethgo.Hash{
									hash1,
								},
							},
						},
					},
				},
			},
		},
	})

	time.Sleep(500 * time.Millisecond)

	m.GetFilterChanges(id)
}

func TestFilter_Block(t *testing.T) {
	store := newMockStore()

	m := NewFilterManager(nil, store)
	go m.Run()

	// add block filter
	id := m.addFilter(nil, nil)

	// emit two events
	store.emitEvent(&mockEvent{
		NewChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: ethgo.HexToHash("1"),
				},
			},
			{
				header: &ethgo.Block{
					Hash: ethgo.HexToHash("2"),
				},
			},
		},
	})

	store.emitEvent(&mockEvent{
		NewChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: ethgo.HexToHash("3"),
				},
			},
		},
	})

	// we need to wait for the manager to process the data
	time.Sleep(500 * time.Millisecond)

	m.GetFilterChanges(id)

	// emit one more event, it should not return the
	// first three hashes
	store.emitEvent(&mockEvent{
		NewChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: ethgo.HexToHash("4"),
				},
			},
		},
	})

	time.Sleep(500 * time.Millisecond)

	m.GetFilterChanges(id)
}

func TestFilter_Timeout(t *testing.T) {
	store := newMockStore()

	m := NewFilterManager(nil, store)
	m.timeout = 2 * time.Second

	go m.Run()

	// add block filter
	id := m.addFilter(nil, nil)

	assert.True(t, m.Exists(id))
	time.Sleep(3 * time.Second)
	assert.False(t, m.Exists(id))
}

func TestFilter_Websocket(t *testing.T) {
	store := newMockStore()

	mock := &mockWsConn{
		msgCh: make(chan []byte, 1),
	}

	m := NewFilterManager(nil, store)
	go m.Run()

	id := m.NewBlockFilter(mock)

	// we cannot call get filter changes for a websocket filter
	_, err := m.GetFilterChanges(id)
	assert.Equal(t, err, errFilterDoesNotExists)

	// emit two events
	store.emitEvent(&mockEvent{
		NewChain: []*mockHeader{
			{
				header: &ethgo.Block{
					Hash: ethgo.HexToHash("1"),
				},
			},
		},
	})

	select {
	case <-mock.msgCh:
	case <-time.After(2 * time.Second):
		t.Fatal("bad")
	}
}

type mockWsConn struct {
	msgCh chan []byte
}

func (m *mockWsConn) WriteMessage(b []byte) error {
	m.msgCh <- b
	return nil
}

func TestFilter_HeadStream(t *testing.T) {
	b := &blockStream{}

	b.push(&ethgo.Block{Hash: ethgo.HexToHash("1")})
	b.push(&ethgo.Block{Hash: ethgo.HexToHash("2")})

	cur := b.Head()

	b.push(&ethgo.Block{Hash: ethgo.HexToHash("3")})
	b.push(&ethgo.Block{Hash: ethgo.HexToHash("4")})

	// get the updates, there are two new entries
	updates, next := cur.getUpdates()

	assert.Equal(t, updates[0].Hash.String(), ethgo.HexToHash("3").String())
	assert.Equal(t, updates[1].Hash.String(), ethgo.HexToHash("4").String())

	// there are no new entries
	updates, _ = next.getUpdates()
	assert.Len(t, updates, 0)
}

type mockStore struct {
	nullBlockchainInterface

	header       *ethgo.Block
	subscription *MockSubscription
	receiptsLock sync.Mutex
	receipts     map[ethgo.Hash][]*ethgo.Receipt
}

func (m *mockStore) GetAccount(root ethgo.Hash, addr ethgo.Address) (*Account, bool, error) {
	panic("implement me")
}

func (m *mockStore) GetStorage(root ethgo.Hash, addr ethgo.Address, slot ethgo.Hash) ([]byte, bool, error) {
	panic("implement me")
}

func (m *mockStore) GetCode(hash ethgo.Hash) ([]byte, error) {
	panic("implement me")
}

func newMockStore() *mockStore {
	return &mockStore{
		header:       &ethgo.Block{Number: 0},
		subscription: NewMockSubscription(),
	}
}

type mockHeader struct {
	header   *ethgo.Block
	receipts []*ethgo.Receipt
}

type mockEvent struct {
	OldChain []*mockHeader
	NewChain []*mockHeader
}

func (m *mockStore) emitEvent(evnt *mockEvent) {
	if m.receipts == nil {
		m.receipts = map[ethgo.Hash][]*ethgo.Receipt{}
	}

	bEvnt := &Event{
		NewChain: []*ethgo.Block{},
		OldChain: []*ethgo.Block{},
	}
	for _, i := range evnt.NewChain {
		m.receipts[i.header.Hash] = i.receipts
		bEvnt.NewChain = append(bEvnt.NewChain, i.header)
	}
	for _, i := range evnt.OldChain {
		m.receipts[i.header.Hash] = i.receipts
		bEvnt.OldChain = append(bEvnt.OldChain, i.header)
	}
	m.subscription.Push(bEvnt)
}

func (m *mockStore) Header() *ethgo.Block {
	return m.header
}

func (m *mockStore) GetReceiptsByHash(hash ethgo.Hash) ([]*ethgo.Receipt, error) {
	m.receiptsLock.Lock()
	defer m.receiptsLock.Unlock()

	receipts := m.receipts[hash]
	return receipts, nil
}

// Subscribe subscribes for chain head events
func (m *mockStore) SubscribeEvents() Subscription {
	return m.subscription
}
