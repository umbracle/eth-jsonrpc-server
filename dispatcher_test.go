package jsonrpc

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/ethgo"
)

var defaultNullLogger = log.New(ioutil.Discard, "", 0)

type mockService struct {
	msgCh chan interface{}
}

func (m *mockService) send(i interface{}) {
	m.msgCh <- i
}

func (m *mockService) Block(f BlockNumber) (interface{}, error) {
	m.send(f)
	return nil, nil
}

func (m *mockService) Type(addr ethgo.Address) (interface{}, error) {
	m.send(addr)
	return nil, nil
}

func (m *mockService) BlockPtr(a string, f *BlockNumber) (interface{}, error) {
	if f == nil {
		m.send(nil)
	} else {
		m.send(*f)
	}
	return nil, nil
}

func (m *mockService) Filter(f LogFilter) (interface{}, error) {
	m.send(f)
	return nil, nil
}

func TestDispatcher_Register(t *testing.T) {
	store := &mockService{}

	s := &Dispatcher{logger: defaultNullLogger}
	s.Register("mock", store)
}

func TestDispatcher_Decoder(t *testing.T) {
	store := &mockService{msgCh: make(chan interface{}, 10)}

	s := &Dispatcher{logger: defaultNullLogger}
	s.Register("mock", store)

	handleReq := func(typ string, msg string) interface{} {
		_, err := s.handleReq(Request{
			Method: "mock_" + typ,
			Params: []byte(msg),
		})
		assert.NoError(t, err)
		return <-store.msgCh
	}

	cases := []struct {
		typ string
		msg string
		res interface{}
	}{
		{
			"block", `["earliest"]`, EarliestBlockNumber,
		},
		{
			"block", `["latest"]`, LatestBlockNumber,
		},
		{
			"block", `["0x1"]`, BlockNumber(1),
		},
		{
			"type", `["` + addr1.String() + `"]`, addr1,
		},
		{
			"blockPtr", `["a"]`, nil,
		},
		{
			"blockPtr", `["a", "latest"]`, LatestBlockNumber,
		},
		{
			"filter", `[{"fromBlock": "pending", "toBlock": "earliest"}]`,
			LogFilter{fromBlock: PendingBlockNumber, toBlock: EarliestBlockNumber},
		},
	}
	for _, c := range cases {
		res := handleReq(c.typ, c.msg)
		if !reflect.DeepEqual(res, c.res) {
			t.Fatal("bad")
		}
	}
}
