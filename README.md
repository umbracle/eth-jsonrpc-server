
# Eth-jsonrpc-server

## Usage

```
package main

import (
    ethjsonrpc "github.com/umbracle/eth-jsonrpc-server"
    "github.com/umbracle/eth-jsonrpc-server/jsonrpc"
)

func main() {
    // Create the jsonrpc server
    srv := jsonrpc.NewServer(
        jsonrpc.WithBindAddr("0.0.0.0:8545"),
        jsonrpc.WithIPC("ipc.path"),
    )

    // bind the ethereum endpoints
    srv.Register(ethjsonrpc.NewEth(&backend{}))
}

type backend struct {
    // ...
}
```
