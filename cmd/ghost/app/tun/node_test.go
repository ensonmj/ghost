package tun

// func TestParseProxyNode(t *testing.T) {
// 	n, err := ParseProxyNode("socks://127.0.0.1:8080")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	expNode := ProxyNode{
// 		Addr:       "127.0.0.1:8080",
// 		Scheme:     "socks5",
// 		Transport:  "",
// 		Remote:     "",
// 		User:       nil,
// 		serverName: "127.0.0.1",
// 	}
// 	if n.Addr != expNode.Addr {
// 		t.Errorf("expect %v, but got %v\n", expNode.Addr, n.Addr)
// 	}
// 	if n.Scheme != expNode.Scheme {
// 		t.Errorf("expect %v, but got %v\n", expNode.Scheme, n.Scheme)
// 	}
// 	if n.Transport != expNode.Transport {
// 		t.Errorf("expect %v, but got %v\n", expNode.Transport, n.Transport)
// 	}
// 	if n.Remote != expNode.Remote {
// 		t.Errorf("expect %v, but got %v\n", expNode.Remote, n.Remote)
// 	}
// 	if n.User != expNode.User {
// 		t.Errorf("expect %v, but got %v\n", expNode.User, n.User)
// 	}
// 	if n.serverName != expNode.serverName {
// 		t.Errorf("expect %v, but got %v\n", expNode.serverName, n.serverName)
// 	}
// }