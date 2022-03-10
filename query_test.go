package jsonrpc

import (
	"reflect"
	"testing"

	"github.com/umbracle/ethgo"
)

var (
	addr1 = ethgo.HexToAddress("1")
	addr2 = ethgo.HexToAddress("2")

	hash1 = ethgo.HexToHash("1")
	hash2 = ethgo.HexToHash("2")
	hash3 = ethgo.HexToHash("3")
)

func TestFilterDecode(t *testing.T) {
	cases := []struct {
		str string
		res *LogFilter
	}{
		{
			`{}`,
			&LogFilter{
				fromBlock: LatestBlockNumber,
				toBlock:   LatestBlockNumber,
			},
		},
		{
			`{
				"address": "1"
			}`,
			nil,
		},
		{
			`{
				"address": "` + addr1.String() + `"
			}`,
			&LogFilter{
				fromBlock: LatestBlockNumber,
				toBlock:   LatestBlockNumber,
				Addresses: []ethgo.Address{
					addr1,
				},
			},
		},
		{
			`{
				"address": [
					"` + addr1.String() + `",
					"` + addr2.String() + `"
				]
			}`,
			&LogFilter{
				fromBlock: LatestBlockNumber,
				toBlock:   LatestBlockNumber,
				Addresses: []ethgo.Address{
					addr1,
					addr2,
				},
			},
		},
		{
			`{
				"topics": [
					"` + hash1.String() + `",
					[
						"` + hash1.String() + `"
					],
					[
						"` + hash1.String() + `",
						"` + hash2.String() + `"
					],
					null,
					"` + hash1.String() + `"
				]
			}`,
			&LogFilter{
				fromBlock: LatestBlockNumber,
				toBlock:   LatestBlockNumber,
				Topics: [][]ethgo.Hash{
					{
						hash1,
					},
					{
						hash1,
					},
					{
						hash1,
						hash2,
					},
					{},
					{
						hash1,
					},
				},
			},
		},
		{
			`{
				"fromBlock": "pending",
				"toBlock": "earliest"
			}`,
			&LogFilter{
				fromBlock: PendingBlockNumber,
				toBlock:   EarliestBlockNumber,
			},
		},
		{
			`{
				"blockHash": "` + hash1.String() + `"
			}`,
			&LogFilter{
				BlockHash: &hash1,
				fromBlock: LatestBlockNumber,
				toBlock:   LatestBlockNumber,
			},
		},
	}

	for indx, c := range cases {
		res := &LogFilter{}
		err := res.UnmarshalJSON([]byte(c.str))
		if err != nil && c.res != nil {
			t.Fatal(err)
		}
		if err == nil && c.res == nil {
			t.Fatal("it should fail")
		}
		if c.res != nil {
			if !reflect.DeepEqual(res, c.res) {
				t.Fatalf("bad %d", indx)
			}
		}
	}
}

func TestFilterMatch(t *testing.T) {
	cases := []struct {
		filter LogFilter
		log    *ethgo.Log
		match  bool
	}{
		{
			// correct, exact match
			LogFilter{
				Topics: [][]ethgo.Hash{
					{
						hash1,
					},
				},
			},
			&ethgo.Log{
				Topics: []ethgo.Hash{
					hash1,
				},
			},
			true,
		},
		{
			// bad, the filter has two hashes
			LogFilter{
				Topics: [][]ethgo.Hash{
					{
						hash1,
					},
					{
						hash1,
					},
				},
			},
			&ethgo.Log{
				Topics: []ethgo.Hash{
					hash1,
				},
			},
			false,
		},
		{
			// correct, wildcard in one hash
			LogFilter{
				Topics: [][]ethgo.Hash{
					{},
					{
						hash2,
					},
				},
			},
			&ethgo.Log{
				Topics: []ethgo.Hash{
					hash1,
					hash2,
				},
			},
			true,
		},
		{
			// correct, more topics than in filter
			LogFilter{
				Topics: [][]ethgo.Hash{
					{
						hash1,
					},
					{
						hash2,
					},
				},
			},
			&ethgo.Log{
				Topics: []ethgo.Hash{
					hash1,
					hash2,
					hash3,
				},
			},
			true,
		},
	}

	for indx, c := range cases {
		if c.filter.Match(c.log) != c.match {
			t.Fatalf("bad %d", indx)
		}
	}
}
