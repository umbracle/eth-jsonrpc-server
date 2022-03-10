package jsonrpc

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var (
	defaultHttpAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8545}
)

type serverType int

const (
	serverIPC serverType = iota
	serverHTTP
	serverWS
)

func (s serverType) String() string {
	switch s {
	case serverIPC:
		return "ipc"
	case serverHTTP:
		return "http"
	case serverWS:
		return "ws"
	default:
		panic("BUG: Not expected")
	}
}

// JSONRPC is an API backend
type JSONRPC struct {
	logger     *log.Logger
	config     *Config
	dispatcher dispatcherImpl
}

type dispatcherImpl interface {
	Handle(reqBody []byte) ([]byte, error)
}

type Config struct {
	Addr *net.TCPAddr
}

// NewJSONRPC returns the JsonRPC http server
func NewJSONRPC(logger *log.Logger, config *Config, dispatcher *Dispatcher) (*JSONRPC, error) {
	if config.Addr == nil {
		config.Addr = defaultHttpAddr
	}
	srv := &JSONRPC{
		logger:     logger,
		config:     config,
		dispatcher: dispatcher,
	}

	// start http server
	if err := srv.setupHTTP(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (j *JSONRPC) setupHTTP() error {
	j.logger.Printf("[INFO] http server started: addr=%s", j.config.Addr.String())

	lis, err := net.Listen("tcp", j.config.Addr.String())
	if err != nil {
		return err
	}

	mux := http.DefaultServeMux
	mux.HandleFunc("/", j.handle)
	mux.HandleFunc("/ws", j.handleWs)

	srv := http.Server{
		Handler: mux,
	}
	go func() {
		if err := srv.Serve(lis); err != nil {
			j.logger.Printf("[ERROR] closed http connection: %v", err)
		}
	}()
	return nil
}

type wrapWsConn struct {
	conn *websocket.Conn
}

func (w *wrapWsConn) WriteMessage(b []byte) error {
	return w.conn.WriteMessage(0, b)
}

func (j *JSONRPC) handleWs(w http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer c.Close()

	wrapConn := &wrapWsConn{conn: c}
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		go func() {
			resp, err := j.dispatcher.Handle(message)
			if err != nil {
				wrapConn.WriteMessage(resp)
			} else {
				wrapConn.WriteMessage([]byte(err.Error()))
			}
		}()
	}
}

func (j *JSONRPC) handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if (*req).Method == "OPTIONS" {
		return
	}

	handleErr := func(err error) {
		w.Write([]byte(err.Error()))
	}
	if req.Method == "GET" {
		w.Write([]byte("JSON-RPC"))
		return
	}
	if req.Method != "POST" {
		w.Write([]byte("method " + req.Method + " not allowed"))
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		handleErr(err)
		return
	}
	resp, err := j.dispatcher.Handle(data)
	if err != nil {
		handleErr(err)
		return
	}
	w.Write(resp)
}
