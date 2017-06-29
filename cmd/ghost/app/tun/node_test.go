package tun

// func TestParseProxyNode(t *testing.T) {
// 	var data = []struct {
// 		in  string
// 		out ProxyNode
// 	}{
// 		{"http://127.0.0.1:8080", ProxyNode{RawURL: "http://127.0.0.1:8080"}},
// 		{"socks://127.0.0.1:8080", ProxyNode{RawURL: "socks://127.0.0.1:8080"}},
// 		{"socks5://127.0.0.1:8080", ProxyNode{RawURL: "socks5://127.0.0.1:8080"}},
// 		// {"quic://127.0.0.1:8080", ProxyNode{RawURL: "quic://127.0.0.1:8080"}},
// 	}
// 	for _, d := range data {
// 		n, err := ParseProxyNode(d.in)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if n.RawURL != d.out.RawURL {
// 			t.Errorf("expect %v, but got %v\n", d.out.RawURL, n.RawURL)
// 		}
// 	}
// }
