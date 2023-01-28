package jsonrpc

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth-jsonrpc-server/jsonrpc"
	"github.com/umbracle/ethgo"
)

func TestEth_Register(t *testing.T) {
	eth := NewEth(&nullBlockchainInterface{})
	d := jsonrpc.NewDispatcher()
	d.Register("eth", eth)
}

type mockAccount struct {
	store   *mockAccountStore
	address ethgo.Address
	code    []byte
	account *Account
	storage map[ethgo.Hash]ethgo.Hash
}

func (m *mockAccount) Storage(k, v ethgo.Hash) {
	m.storage[k] = v
}

func (m *mockAccount) Code(code []byte) {
	codeHash := ethgo.BytesToHash(m.address[:])
	m.code = code
	m.account.CodeHash = codeHash[:]
}

func (m *mockAccount) Nonce(n uint64) {
	m.account.Nonce = n
}

func (m *mockAccount) Balance(n uint64) {
	m.account.Balance = new(big.Int).SetUint64(n)
}

type mockAccountStore struct {
	nullBlockchainInterface
	accounts map[ethgo.Address]*mockAccount
}

func (m *mockAccountStore) AddAccount(addr ethgo.Address) *mockAccount {
	if m.accounts == nil {
		m.accounts = map[ethgo.Address]*mockAccount{}
	}
	acct := &mockAccount{
		store:   m,
		address: addr,
		account: &Account{},
		storage: map[ethgo.Hash]ethgo.Hash{},
	}
	m.accounts[addr] = acct
	return acct
}

func (m *mockAccountStore) Header() *ethgo.Block {
	return &ethgo.Block{}
}

func (m *mockAccountStore) GetAccount(root ethgo.Hash, addr ethgo.Address) (*Account, bool, error) {
	acct, ok := m.accounts[addr]
	if !ok {
		return nil, false, nil
	}
	return acct.account, true, nil
}

func (m *mockAccountStore) GetCode(hash ethgo.Hash) ([]byte, error) {
	for _, acct := range m.accounts {
		if bytes.Equal(acct.account.CodeHash, hash[:]) {
			return acct.code, nil
		}
	}
	return nil, fmt.Errorf("code not found")
}

func (m *mockAccountStore) GetStorage(root ethgo.Hash, addr ethgo.Address, slot ethgo.Hash) ([]byte, bool, error) {
	acct := m.accounts[addr]

	val, ok := acct.storage[slot]
	if !ok {
		return nil, false, nil
	}
	return val[:], true, nil
}

type mockBlockStore struct {
	nullBlockchainInterface
	blocks []*ethgo.Block
}

func (m *mockBlockStore) add(blocks ...*ethgo.Block) {
	if m.blocks == nil {
		m.blocks = []*ethgo.Block{}
	}
	m.blocks = append(m.blocks, blocks...)
}

func (m *mockBlockStore) GetReceiptsByHash(hash ethgo.Hash) ([]*ethgo.Receipt, error) {
	return nil, nil
}

func (m *mockBlockStore) GetHeaderByNumber(blockNumber uint64) (*ethgo.Block, bool) {
	b, ok := m.GetBlockByNumber(blockNumber, false)
	if !ok {
		return nil, false
	}
	return b, true
}

func (m *mockBlockStore) GetBlockByNumber(blockNumber uint64, full bool) (*ethgo.Block, bool) {
	for _, b := range m.blocks {
		if b.Number == blockNumber {
			return b, true
		}
	}
	return nil, false
}

func (m *mockBlockStore) GetBlockByHash(hash ethgo.Hash, full bool) (*ethgo.Block, bool) {
	for _, b := range m.blocks {
		if b.Hash == hash {
			return b, true
		}
	}
	return nil, false
}

func (m *mockBlockStore) Header() *ethgo.Block {
	return m.blocks[len(m.blocks)-1]
}

func TestEth_Block_GetBlockByNumber(t *testing.T) {
	b := &mockBlockStore{}
	for i := 0; i < 10; i++ {
		b.add(&ethgo.Block{
			Number: uint64(i),
		})
	}

	eth := NewEth(b)
	getBlockByNumber := func(num BlockNumber) bool {
		_, err := eth.GetBlockByNumber(num, false)
		return err == nil
	}

	// tag is latest block
	assert.True(t, getBlockByNumber(LatestBlockNumber))

	// earliest header is not supported
	assert.False(t, getBlockByNumber(EarliestBlockNumber))

	// block number is negative
	assert.False(t, getBlockByNumber(BlockNumber(-50)))

	// block is genesis
	assert.True(t, getBlockByNumber(BlockNumber(0)))

	// block number in range
	assert.True(t, getBlockByNumber(BlockNumber(2)))

	// block number is too big
	assert.False(t, getBlockByNumber(BlockNumber(50)))
}

func TestEth_Block_GetBlockByHash(t *testing.T) {
	b := &mockBlockStore{}
	b.add(&ethgo.Block{
		Hash: hash1,
	})

	eth := NewEth(b)

	_, err := eth.GetBlockByHash(hash1, false)
	assert.NoError(t, err)

	_, err = eth.GetBlockByHash(hash2, false)
	assert.Error(t, err)
}

func TestEth_Block_BlockNumber(t *testing.T) {
	b := &mockBlockStore{}
	b.add(&ethgo.Block{
		Number: 10,
	})

	eth := NewEth(b)

	num, err := eth.BlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, num.Uint64(), uint64(10))
}

type mockStoreLogs struct {
	nullBlockchainInterface
	receipts map[ethgo.Hash][]*ethgo.Receipt
	input    *GetLogsInput
}

func (m *mockStoreLogs) addReceipt(hash ethgo.Hash, receipts []*ethgo.Receipt) {
	if len(m.receipts) == 0 {
		m.receipts = map[ethgo.Hash][]*ethgo.Receipt{}
	}
	m.receipts[hash] = receipts
}

func (m *mockStoreLogs) GetReceiptsByHash(hash ethgo.Hash) ([]*ethgo.Receipt, error) {
	for h, r := range m.receipts {
		if h == hash {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockStoreLogs) GetLogs(input *GetLogsInput) ([]*ethgo.Log, error) {
	m.input = input
	return nil, nil
}

func TestEth_Logs_BlockHash(t *testing.T) {
	matchReceipt := &ethgo.Receipt{
		Logs: []*ethgo.Log{
			{
				Address: addr1,
				Topics:  []ethgo.Hash{hash2, hash3},
			},
		},
	}

	b := &mockStoreLogs{}
	b.addReceipt(hash1, []*ethgo.Receipt{
		matchReceipt,
	})
	b.addReceipt(hash2, []*ethgo.Receipt{
		matchReceipt,
	})

	eth := NewEth(b)

	filter := &LogFilter{
		BlockHash: &hash1,
		Topics: [][]ethgo.Hash{
			{hash2},
		},
	}
	logs, err := eth.GetLogs(filter)
	assert.NoError(t, err)
	assert.Equal(t, logs[0].Address, addr1)

	logs, err = eth.GetLogs(&LogFilter{BlockHash: &hash3})
	assert.NoError(t, err)
	assert.Empty(t, logs)
}

func TestEth_Logs_BlockRange(t *testing.T) {
	b := &mockStoreLogs{}

	eth := NewEth(b)

	_, err := eth.GetLogs(&LogFilter{fromBlock: 10, toBlock: 15})
	assert.NoError(t, err)
}

var (
	addr0 = ethgo.Address{0x1}
)

func TestEth_State_GetBalance(t *testing.T) {
	store := &mockAccountStore{}

	acct0 := store.AddAccount(addr0)
	acct0.Balance(100)

	eth := NewEth(store)

	balance, err := eth.GetBalance(addr0, LatestBlockNumber)
	assert.NoError(t, err)
	assert.Equal(t, balance, argBigPtr(big.NewInt(100)))
}

func TestEth_State_GetTransactionCount(t *testing.T) {
	store := &mockAccountStore{}

	acct0 := store.AddAccount(addr0)
	acct0.Nonce(100)

	eth := NewEth(store)

	balance, err := eth.GetTransactionCount(addr0, LatestBlockNumber)
	assert.NoError(t, err)
	assert.Equal(t, balance, argUintPtr(100))
}

func TestEth_State_GetCode(t *testing.T) {
	store := &mockAccountStore{}

	code0 := []byte{0x1, 0x2, 0x3}
	acct0 := store.AddAccount(addr0)
	acct0.Code(code0)

	eth := NewEth(store)

	// get code of known account
	code, err := eth.GetCode(addr0, LatestBlockNumber)
	assert.NoError(t, err)
	assert.Equal(t, code.Bytes(), code0)
}

func TestEth_State_GetStorageAt(t *testing.T) {
	store := &mockAccountStore{}

	acct0 := store.AddAccount(addr0)
	acct0.Storage(hash1, hash1)

	eth := NewEth(store)

	res, err := eth.GetStorageAt(acct0.address, hash1, LatestBlockNumber)
	assert.NoError(t, err)
	assert.Equal(t, res, argBytesPtr(hash1[:]))
}

type mockStoreTxn struct {
	nullBlockchainInterface

	txn []byte
}

func (m *mockStoreTxn) AddTx(tx []byte) (ethgo.Hash, error) {
	m.txn = tx
	return ethgo.Hash{0x1}, nil
}

func (m *mockStoreTxn) GetNonce(addr ethgo.Address) (uint64, bool) {
	return 1, false
}

func TestEth_TxnPool_SendRawTransaction(t *testing.T) {
	b := &mockStoreTxn{}
	eth := NewEth(b)

	hash, err := eth.SendRawTransaction(argBytes([]byte{0x1}))
	assert.NoError(t, err)

	assert.Equal(t, hash.Bytes(), ethgo.Hash{0x1}.Bytes())
}
