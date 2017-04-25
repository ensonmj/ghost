package tun

import "testing"

func TestParseProxyNode(t *testing.T) {
	n, err := ParseProxyNode("socks://127.0.0.1:8080")
	if err != nil {
		t.Error(err)
	}
	expNode := {
		Addr: "127.0.0.1:8080",
		Protocol: "socks",
		Transport: "socks",
		Remote: "",
		User: "",
		serverName:"127.0.0.1",
	}
	if n != expNode {
		t.Errorf("expect %v, but got %v\n", expNode, n)
	}
}
