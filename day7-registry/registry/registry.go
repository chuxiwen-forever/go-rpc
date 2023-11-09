package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
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

func (r *GoRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now()
	}
}

func (r *GoRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *GoRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Gorpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Gorpc-server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GoRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGoRegister.HandleHTTP(defaultPath)
}
