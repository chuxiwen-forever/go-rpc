package registry

import (
	"sync"
	"time"
)

type ServerItem struct {
	Addr  string
	start time.Time
}

type GoRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

const (
	defaultPath    = "/_gorpc_/registry"
	defaultTimeout = time.Minute * 5
)

func New(timeout time.Duration) *GoRegistry {
	return &GoRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultGoRegister = New(defaultTimeout)
