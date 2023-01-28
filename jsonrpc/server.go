package jsonrpc

import (
	"io/ioutil"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

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

type Server struct {
	config     *Config
	dispatcher dispatcherImpl
}

type dispatcherImpl interface {
	Handle(reqBody []byte) ([]byte, error)
}

func NewServer(opts ...ConfigOption) (*Server, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	srv := &Server{
		config:     config,
		dispatcher: NewDispatcher(),
	}

	// start http server
	if err := srv.setupHTTP(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (j *Server) setupHTTP() error {
	addr, err := net.ResolveTCPAddr("tcp", j.config.Addr)
	if err != nil {
		return err
	}

	j.config.Logger.Printf("[INFO] http server started: addr=%s", addr.String())

	lis, err := net.Listen("tcp", addr.String())
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
			j.config.Logger.Printf("[ERROR] closed http connection: %v", err)
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

func (j *Server) handleWs(w http.ResponseWriter, req *http.Request) {
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

func (j *Server) handle(w http.ResponseWriter, req *http.Request) {
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
