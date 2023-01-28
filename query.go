package jsonrpc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/umbracle/ethgo"
)

// LogFilter is a filter for logs
type LogFilter struct {
	BlockHash *ethgo.Hash

	fromBlock BlockNumber
	toBlock   BlockNumber

	Addresses []ethgo.Address
	Topics    [][]ethgo.Hash
}

// addTopicSet adds specific topics to the log filter topics
func (l *LogFilter) addTopicSet(set ...string) error {
	if l.Topics == nil {
		l.Topics = [][]ethgo.Hash{}
	}

	res := []ethgo.Hash{}
	for _, i := range set {
		item := ethgo.Hash{}
		if err := item.UnmarshalText([]byte(i)); err != nil {
			return err
		}
		res = append(res, item)
	}

	l.Topics = append(l.Topics, res)
	return nil
}

// addAddress Adds the address to the log filter
func (l *LogFilter) addAddress(raw string) error {
	if l.Addresses == nil {
		l.Addresses = []ethgo.Address{}
	}
	addr := ethgo.Address{}
	if err := addr.UnmarshalText([]byte(raw)); err != nil {
		return err
	}

	l.Addresses = append(l.Addresses, addr)
	return nil
}

// UnmarshalJSON decodes a json object
func (l *LogFilter) UnmarshalJSON(data []byte) error {
	var obj struct {
		BlockHash *ethgo.Hash   `json:"blockHash"`
		FromBlock string        `json:"fromBlock"`
		ToBlock   string        `json:"toBlock"`
		Address   interface{}   `json:"address"`
		Topics    []interface{} `json:"topics"`
	}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}

	l.BlockHash = obj.BlockHash

	if obj.FromBlock == "" {
		l.fromBlock = LatestBlockNumber
	} else {
		if l.fromBlock, err = stringToBlockNumber(obj.FromBlock); err != nil {
			return err
		}
	}

	if obj.ToBlock == "" {
		l.toBlock = LatestBlockNumber
	} else {
		if l.toBlock, err = stringToBlockNumber(obj.ToBlock); err != nil {
			return err
		}
	}

	if obj.Address != nil {
		// decode address, either "" or [""]
		switch raw := obj.Address.(type) {
		case string:
			// ""
			if err := l.addAddress(raw); err != nil {
				return err
			}

		case []interface{}:
			// ["", ""]
			for _, addr := range raw {
				if item, ok := addr.(string); ok {
					if err := l.addAddress(item); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("address expected")
				}
			}

		default:
			return fmt.Errorf("failed to decode address. Expected either '' or ['', '']")
		}
	}

	if obj.Topics != nil {
		// decode topics, either "" or ["", ""] or null
		for _, item := range obj.Topics {
			switch raw := item.(type) {
			case string:
				// ""
				if err := l.addTopicSet(raw); err != nil {
					return err
				}

			case []interface{}:
				// ["", ""]
				res := []string{}
				for _, i := range raw {
					if item, ok := i.(string); ok {
						res = append(res, item)
					} else {
						return fmt.Errorf("hash expected")
					}
				}
				if err := l.addTopicSet(res...); err != nil {
					return err
				}

			case nil:
				// null
				if err := l.addTopicSet(); err != nil {
					return err
				}

			default:
				return fmt.Errorf("failed to decode topics. Expected '' or [''] or null")
			}
		}
	}

	// decode topics
	return nil
}

// Match returns whether the receipt includes topics for this filter
func (l *LogFilter) Match(log *ethgo.Log) bool {
	// check addresses
	if len(l.Addresses) > 0 {
		match := false
		for _, addr := range l.Addresses {
			if addr == log.Address {
				match = true
			}
		}
		if !match {
			return false
		}
	}
	// check topics
	if len(l.Topics) > len(log.Topics) {
		return false
	}
	for i, sub := range l.Topics {
		match := len(sub) == 0
		for _, topic := range sub {
			if log.Topics[i] == topic {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

const (
	PendingBlockNumber  = BlockNumber(-3)
	LatestBlockNumber   = BlockNumber(-2)
	EarliestBlockNumber = BlockNumber(-1)
)

type BlockNumber int64

func stringToBlockNumber(str string) (BlockNumber, error) {
	if str == "" {
		return 0, fmt.Errorf("value is empty")
	}

	str = strings.Trim(str, "\"")
	switch str {
	case "pending":
		return PendingBlockNumber, nil
	case "latest":
		return LatestBlockNumber, nil
	case "earliest":
		return EarliestBlockNumber, nil
	}

	n, err := parseUint64orHex(&str)
	if err != nil {
		return 0, err
	}
	return BlockNumber(n), nil
}

// UnmarshalJSON automatically decodes the user input for the block number, when a JSON RPC method is called
func (b *BlockNumber) UnmarshalJSON(buffer []byte) error {
	num, err := stringToBlockNumber(string(buffer))
	if err != nil {
		return err
	}
	*b = num
	return nil
}

func parseUint64orHex(val *string) (uint64, error) {
	if val == nil {
		return 0, nil
	}

	str := *val
	base := 10
	if strings.HasPrefix(str, "0x") {
		str = str[2:]
		base = 16
	}
	return strconv.ParseUint(str, base, 64)
}
