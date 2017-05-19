package tun

import "testing"

func TestParseProxyNode(t *testing.T) {
	n, err := ParseProxyNode("socks://127.0.0.1:8080")
	if err != nil {
		t.Error(err)
	}
	expNode := ProxyNode{
		RawURL: "socks://127.0.0.1:8080",
	}
	if n.RawURL != expNode.RawURL {
		t.Errorf("expect %v, but got %v\n", expNode.RawURL, n.RawURL)
	}
}
