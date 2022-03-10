package jsonrpc

import (
	"fmt"
	"math/big"

	"github.com/umbracle/ethgo"
)

type EthBackend interface {
	blockchainInterface
}

// Eth is the eth jsonrpc endpoint
type Eth struct {
	f *FilterManager
	b EthBackend
}

func NewEth(b EthBackend) *Eth {
	e := &Eth{
		b: b,
		f: NewFilterManager(nil, b),
	}
	go e.f.Run()
	return e
}

// ChainId returns the chain id of the client
func (e *Eth) ChainId() (interface{}, error) {
	return argUintPtr(e.b.ChainID()), nil
}

// GetBlockByNumber returns information about a block by block number
func (e *Eth) GetBlockByNumber(number BlockNumber, full bool) (*ethgo.Block, error) {
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return nil, err
	}
	return header, nil
}

// GetBlockByHash returns information about a block by hash
func (e *Eth) GetBlockByHash(hash ethgo.Hash, full bool) (*ethgo.Block, error) {
	block, ok := e.b.GetBlockByHash(hash, full)
	if !ok {
		return nil, fmt.Errorf("unable to get block by hash %v", hash)
	}
	return block, nil
}

// BlockNumber returns current block number
func (e *Eth) BlockNumber() (argUint64, error) {
	h := e.b.Header()
	if h == nil {
		return argUint64(0), fmt.Errorf("header has a nil value")
	}
	return argUint64(h.Number), nil
}

// SendRawTransaction sends a raw transaction
func (e *Eth) SendRawTransaction(input argBytes) (argBytes, error) {
	tx := &ethgo.Transaction{}
	if err := tx.UnmarshalRLP(input); err != nil {
		return nil, err
	}
	hash, err := e.b.AddTx(tx)
	if err != nil {
		return nil, err
	}
	return argBytes(hash[:]), nil
}

// GetTransactionByHash returns a transaction by his hash
func (e *Eth) GetTransactionByHash(hash ethgo.Hash) (interface{}, error) {
	txn, err := e.b.GetTransactionByHash(hash)
	if err != nil {
		// txn not found
		return nil, err
	}
	if txn == nil {
		return nil, nil
	}
	return txn.Transaction, nil
}

// GetTransactionReceipt returns a transaction receipt by his hash
func (e *Eth) GetTransactionReceipt(hash ethgo.Hash) (interface{}, error) {
	txn, err := e.b.GetTransactionByHash(hash)
	if err != nil {
		// txn not found
		return nil, err
	}
	if txn == nil {
		return nil, nil
	}
	if txn.Receipt == nil {
		return nil, nil
	}
	return txn.Receipt, nil
}

// GetStorageAt returns the contract storage at the index position
func (e *Eth) GetStorageAt(address ethgo.Address, index ethgo.Hash, number BlockNumber) (interface{}, error) {
	// Fetch the requested header
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return nil, err
	}

	// Get the storage for the passed in location
	result, err := e.b.GetStorage(header.StateRoot, address, index)
	if err != nil {
		return nil, err
	}
	return argBytesPtr(result), nil
}

// GasPrice returns the average gas price based on the last x blocks
func (e *Eth) GasPrice() (interface{}, error) {
	return argBigPtr(e.b.GetAvgGasPrice()), nil
}

// Call executes a smart contract call using the transaction object data
func (e *Eth) Call(arg *txnArgs, number BlockNumber) (interface{}, error) {
	transaction, err := e.decodeTxn(arg)
	if err != nil {
		return nil, err
	}
	// Fetch the requested header
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return nil, err
	}

	retValue, err := e.b.Call(transaction, header)
	if err != nil {
		return nil, err
	}
	return argBytesPtr(retValue), nil
}

// EstimateGas estimates the gas needed to execute a transaction
func (e *Eth) EstimateGas(arg *txnArgs, rawNum *BlockNumber) (interface{}, error) {
	transaction, err := e.decodeTxn(arg)
	if err != nil {
		return nil, err
	}
	gas, err := e.b.EstimateGas(transaction, nil)
	if err != nil {
		return nil, err
	}
	return argUint64(gas), nil
}

// GetLogs returns an array of logs matching the filter options
func (e *Eth) GetLogs(filterOptions *LogFilter) ([]*ethgo.Log, error) {
	head := e.b.Header()

	if filterOptions.BlockHash != nil {
		receipts, err := e.b.GetReceiptsByHash(*filterOptions.BlockHash)
		if err != nil {
			return nil, err
		}
		var result []*ethgo.Log
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				if filterOptions.Match(log) {
					result = append(result, log)
				}
			}
		}
		return result, nil
	}

	resolveNum := func(num BlockNumber) uint64 {
		if num == PendingBlockNumber || num == EarliestBlockNumber {
			num = LatestBlockNumber
		}
		if num == LatestBlockNumber {
			return head.Number
		}
		return uint64(num)
	}

	from := resolveNum(filterOptions.fromBlock)
	to := resolveNum(filterOptions.toBlock)

	if to < from {
		return nil, fmt.Errorf("incorrect range")
	}

	input := &GetLogsInput{
		From:      from,
		To:        to,
		Addresses: filterOptions.Addresses,
		Topics:    filterOptions.Topics,
	}
	logs, err := e.b.GetLogs(input)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetBalance returns the account's balance at the referenced block
func (e *Eth) GetBalance(address ethgo.Address, number BlockNumber) (*argBig, error) {
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return nil, err
	}

	acc, err := e.b.GetAccount(header.StateRoot, address)
	if acc == nil || err != nil {
		return nil, err
	}
	return argBigPtr(acc.Balance), nil
}

// GetTransactionCount returns account nonce
func (e *Eth) GetTransactionCount(address ethgo.Address, number BlockNumber) (interface{}, error) {
	nonce, err := e.getNextNonce(address, number)
	if err != nil {
		return nil, err
	}
	return argUintPtr(nonce), nil
}

// GetCode returns account code at given block number
func (e *Eth) GetCode(address ethgo.Address, number BlockNumber) (argBytes, error) {
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return nil, err
	}
	acc, err := e.b.GetAccount(header.StateRoot, address)
	if acc == nil || err != nil {
		return nil, err
	}
	return e.b.GetCode(ethgo.BytesToHash(acc.CodeHash))
}

// NewFilter creates a filter object, based on filter options, to notify when the state changes (logs).
func (e *Eth) NewFilter(filter *LogFilter) (interface{}, error) {
	return e.f.NewLogFilter(filter, nil), nil
}

// NewBlockFilter creates a filter in the node, to notify when a new block arrives
func (e *Eth) NewBlockFilter() (interface{}, error) {
	return e.f.NewBlockFilter(nil), nil
}

// GetFilterChanges is a polling method for a filter, which returns an array of logs which occurred since last poll.
func (e *Eth) GetFilterChanges(id string) (interface{}, error) {
	return e.f.GetFilterChanges(id)
}

// UninstallFilter uninstalls a filter with given ID
func (e *Eth) UninstallFilter(id string) (bool, error) {
	ok := e.f.Uninstall(id)
	return ok, nil
}

// Unsubscribe uninstalls a filter in a websocket
func (e *Eth) Unsubscribe(id string) (bool, error) {
	ok := e.f.Uninstall(id)
	return ok, nil
}

func (e *Eth) getBlockHeaderImpl(number BlockNumber) (*ethgo.Block, error) {
	switch number {
	case LatestBlockNumber:
		return e.b.Header(), nil

	case EarliestBlockNumber:
		return nil, fmt.Errorf("fetching the earliest header is not supported")

	case PendingBlockNumber:
		return nil, fmt.Errorf("fetching the pending header is not supported")

	default:
		// Convert the block number from hex to uint64
		header, ok := e.b.GetHeaderByNumber(uint64(number))
		if !ok {
			return nil, fmt.Errorf("Error fetching block number %d header", uint64(number))
		}
		return header, nil
	}
}

func (e *Eth) getNextNonce(address ethgo.Address, number BlockNumber) (uint64, error) {
	if number == PendingBlockNumber {
		res, ok := e.b.GetNonce(address)
		if ok {
			return res, nil
		}
		number = LatestBlockNumber
	}
	header, err := e.getBlockHeaderImpl(number)
	if err != nil {
		return 0, err
	}
	acc, err := e.b.GetAccount(header.StateRoot, address)
	if err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

func (e *Eth) decodeTxn(arg *txnArgs) (*ethgo.Transaction, error) {
	// set default values
	if arg.From == nil {
		return nil, fmt.Errorf("from is empty")
	}
	if arg.Data != nil && arg.Input != nil {
		return nil, fmt.Errorf("both input and data cannot be set")
	}
	if arg.Nonce == nil {
		// get nonce from the pool
		nonce, err := e.getNextNonce(*arg.From, LatestBlockNumber)
		if err != nil {
			return nil, err
		}
		arg.Nonce = argUintPtr(nonce)
	}
	if arg.Value == nil {
		arg.Value = argBytesPtr([]byte{})
	}
	if arg.GasPrice == nil {
		// use the suggested gas price
		arg.GasPrice = argBytesPtr(e.b.GetAvgGasPrice().Bytes())
	}

	var input []byte
	if arg.Data != nil {
		input = *arg.Data
	} else if arg.Input != nil {
		input = *arg.Input
	}
	if arg.To == nil {
		if input == nil {
			return nil, fmt.Errorf("contract creation without data provided")
		}
	}
	if input == nil {
		input = []byte{}
	}

	if arg.Gas == nil {
		// TODO
		arg.Gas = argUintPtr(1000000)
	}

	txn := &ethgo.Transaction{
		From: *arg.From,
		Gas:  uint64(*arg.Gas),
		// GasPrice: new(big.Int).SetBytes(*arg.GasPrice),
		Value: new(big.Int).SetBytes(*arg.Value),
		Input: input,
		Nonce: uint64(*arg.Nonce),
	}
	if arg.To != nil {
		txn.To = arg.To
	}

	hash, err := txn.GetHash()
	if err != nil {
		return nil, err
	}
	txn.Hash = hash

	return txn, nil
}
