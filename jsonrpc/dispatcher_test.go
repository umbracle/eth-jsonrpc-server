package jsonrpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDispatcher_Register(t *testing.T) {
	srv := &mockService{}

	d := NewDispatcher()
	d.Register("mock", srv)

	cases := []struct {
		method string
		err    bool
		result string
	}{
		{
			method: "mock_str",
			result: "\"a\"",
		},
		{
			method: "mock_num",
			result: "1",
		},
		{
			method: "mock_err",
			err:    true,
		},
	}

	for _, c := range cases {
		resp, err := d.handleReq(Request{
			Method: c.method,
		})
		if c.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, string(resp.Result), c.result)
		}
	}
}

type mockService struct {
}

func (m *mockService) Str() (string, error) {
	return "a", nil
}

func (m *mockService) Num() (uint64, error) {
	return 1, nil
}

func (m *mockService) Err() (interface{}, error) {
	return nil, fmt.Errorf("err")
}
