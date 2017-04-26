package tun

import (
	"testing"
)

func TestNewProxyChain(t *testing.T) {
	nodes := []string{
		"socks://127.0.0.1:8080",
		"http://127.0.0.1",
	}
	chain, err := NewProxyChain(nodes...)
	if err != nil {
		t.Error(err)
	}
	if len(chain.nodes) != 2 {
		t.Errorf("expect %d nodes, but got %d\n", 2, len(chain.nodes))
	}
}
