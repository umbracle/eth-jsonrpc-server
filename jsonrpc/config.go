package jsonrpc

import (
	"io/ioutil"
	"log"
)

type Config struct {
	Addr         string
	Logger       *log.Logger
	IpcPath      string
	MaxBatchSize uint64
}

type ConfigOption func(*Config)

func WithBindAddr(addr string) ConfigOption {
	return func(h *Config) {
		h.Addr = addr
	}
}

func WithIPC(ipcPath string) ConfigOption {
	return func(h *Config) {
		h.IpcPath = ipcPath
	}
}

func WithLogger(logger *log.Logger) ConfigOption {
	return func(h *Config) {
		h.Logger = logger
	}
}

func WithMaxBatchSize(maxBatchSize uint64) ConfigOption {
	return func(h *Config) {
		h.MaxBatchSize = maxBatchSize
	}
}

func DefaultConfig() *Config {
	return &Config{
		Logger: log.New(ioutil.Discard, "", 0),
	}
}
