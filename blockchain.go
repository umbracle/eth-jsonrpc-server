package jsonrpc

import (
	"math/big"

	"github.com/umbracle/ethgo"
)

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     ethgo.Hash
	CodeHash []byte
}

// Subscription is the blockchain subscription interface
type Subscription interface {
	GetEvent() *Event
	Close()
}

type EventType int

const (
	// New head event
	EventHead EventType = iota

	// Chain reorganization event
	EventReorg

	// Chain fork event
	EventFork
)

// Event is the blockchain event that gets passed to the listeners
type Event struct {
	// Old chain (removed headers) if there was a reorg
	OldChain []*ethgo.Block

	// New part of the chain (or a fork)
	NewChain []*ethgo.Block

	// Type is the type of event
	Type EventType
}

// blockchain is the interface with the blockchain required
// by the filter manager
type blockchainInterface interface {
	// ChainID returns the chain id of the blockchain
	ChainID() uint64

	// Header returns the current header of the chain (genesis if empty)
	Header() *ethgo.Block

	// GetReceiptsByHash returns the receipts for a block hash
	GetReceiptsByHash(hash ethgo.Hash) ([]*ethgo.Receipt, error)

	// EstimateGas estimates the gas to run the transactio
	EstimateGas(tx *ethgo.Transaction, header *ethgo.Block) (uint64, error)

	// Calls calls the transaction
	Call(tx *ethgo.Transaction, header *ethgo.Block) ([]byte, error)

	// AddTx adds a new transaction to the tx pool
	AddTx(tx []byte) (ethgo.Hash, error)

	// GetTransactionByHash returns a transaction by its hash
	GetTransactionByHash(hash ethgo.Hash) (*TransactionResult, error)

	// SubscribeEvents subscribes for chain head events
	SubscribeEvents() Subscription

	// GetAvgGasPrice returns the average gas price
	GetAvgGasPrice() *big.Int

	// GetBlockByHash gets a block using the provided hash
	GetBlockByHash(hash ethgo.Hash, full bool) (*ethgo.Block, bool)

	// GetBlockByNumber returns a block using the provided number
	GetBlockByNumber(num uint64, full bool) (*ethgo.Block, bool)

	// GetPendingNonce returns the next nonce for this address on the transaction pool
	GetPendingNonce(addr ethgo.Address) (uint64, bool)

	// GetAccount returns the account object for a given address
	GetAccount(root ethgo.Hash, addr ethgo.Address) (*Account, bool, error)

	// GetStorage returns the storage slot for a given address
	GetStorage(root ethgo.Hash, addr ethgo.Address, slot ethgo.Hash) ([]byte, bool, error)

	// GetCode returns a code by its hash
	GetCode(hash ethgo.Hash) ([]byte, error)

	// GetLogs returns an array of logs given some filter input
	GetLogs(input *GetLogsInput) ([]*ethgo.Log, error)
}

type GetLogsInput struct {
	From      uint64
	To        uint64
	Addresses []ethgo.Address
	Topics    [][]ethgo.Hash
}

type TransactionResult struct {
	Transaction *ethgo.Transaction
	Receipt     *ethgo.Receipt
}

type nullBlockchainInterface struct {
}

func (b *nullBlockchainInterface) ChainID() uint64 {
	return 0
}

func (b *nullBlockchainInterface) Header() *ethgo.Block {
	return &ethgo.Block{Number: 0}
}

func (b *nullBlockchainInterface) GetReceiptsByHash(hash ethgo.Hash) ([]*ethgo.Receipt, error) {
	return nil, nil
}

func (b *nullBlockchainInterface) EstimateGas(tx *ethgo.Transaction, header *ethgo.Block) (uint64, error) {
	return 0, nil
}

func (b *nullBlockchainInterface) Call(tx *ethgo.Transaction, header *ethgo.Block) ([]byte, error) {
	return nil, nil
}

func (b *nullBlockchainInterface) AddTx(tx []byte) (ethgo.Hash, error) {
	return ethgo.Hash{}, nil
}

func (b *nullBlockchainInterface) GetTransactionByHash(hash ethgo.Hash) (*TransactionResult, error) {
	return nil, nil
}

func (b *nullBlockchainInterface) SubscribeEvents() Subscription {
	return NewMockSubscription()
}

func (b *nullBlockchainInterface) GetHeaderByNumber(block uint64) (*ethgo.Block, bool) {
	return nil, false
}

func (b *nullBlockchainInterface) GetAvgGasPrice() *big.Int {
	return nil
}

func (b *nullBlockchainInterface) GetBlockByHash(hash ethgo.Hash, full bool) (*ethgo.Block, bool) {
	return nil, false
}

func (b *nullBlockchainInterface) GetBlockByNumber(num uint64, full bool) (*ethgo.Block, bool) {
	return nil, false
}

func (b *nullBlockchainInterface) GetPendingNonce(addr ethgo.Address) (uint64, bool) {
	return 0, false
}

func (b *nullBlockchainInterface) GetCode(hash ethgo.Hash) ([]byte, error) {
	return nil, nil
}

func (b *nullBlockchainInterface) GetStorage(root ethgo.Hash, addr ethgo.Address, slot ethgo.Hash) ([]byte, bool, error) {
	return nil, false, nil
}

func (b *nullBlockchainInterface) GetAccount(root ethgo.Hash, addr ethgo.Address) (*Account, bool, error) {
	return nil, false, nil
}

func (b *nullBlockchainInterface) GetLogs(input *GetLogsInput) ([]*ethgo.Log, error) {
	return nil, nil
}

type MockSubscription struct {
	eventCh chan *Event
}

func NewMockSubscription() *MockSubscription {
	return &MockSubscription{eventCh: make(chan *Event)}
}

func (m *MockSubscription) Push(e *Event) {
	m.eventCh <- e
}

func (m *MockSubscription) GetEventCh() chan *Event {
	return m.eventCh
}

func (m *MockSubscription) GetEvent() *Event {
	evnt := <-m.eventCh
	return evnt
}

func (m *MockSubscription) Close() {
}
