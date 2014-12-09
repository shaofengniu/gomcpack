package npc

import (
	"errors"
	"math/rand"
	"net"
	"strings"
	"sync"
)

var ErrNoServers = errors.New("no server configured or available")

type ServerSelector interface {
	PickServer() (net.Addr, error)
}

type ServerList struct {
	sync.RWMutex
	addrs []net.Addr
}

func (ss *ServerList) SetServers(servers []string) error {
	naddr := make([]net.Addr, len(servers))
	for i, server := range servers {
		if strings.Contains(server, "/") {
			addr, err := net.ResolveUnixAddr("unix", server)
			if err != nil {
				return err
			}
			naddr[i] = addr
		} else {
			tcpaddr, err := net.ResolveTCPAddr("tcp", server)
			if err != nil {
				return err
			}
			naddr[i] = tcpaddr
		}
	}
	ss.Lock()
	ss.addrs = naddr
	ss.Unlock()
	return nil
}

func (ss *ServerList) PickServer() (net.Addr, error) {
	ss.RLock()
	defer ss.RUnlock()
	if len(ss.addrs) == 0 {
		return nil, ErrNoServers
	}
	if len(ss.addrs) == 1 {
		return ss.addrs[0], nil
	}
	return ss.addrs[rand.Int63n(int64(len(ss.addrs)))], nil
}
