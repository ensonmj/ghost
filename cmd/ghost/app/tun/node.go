package tun

import (
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type ProxyNode struct {
	URL    url.URL
	RawURL string
}

func (n ProxyNode) String() string {
	return n.URL.String()
}

// The proxy node string pattern is [scheme://][user:pass@host]:port.
func ParseProxyNode(rawurl string) (*ProxyNode, error) {
	if !strings.Contains(rawurl, "://") {
		rawurl = "http://" + rawurl
	}

	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, errors.Wrap(err, "proxy node parse")
	}

	// http/https/http2/socks5/tcp/udp/rtcp/rudp/ss/ws/wss
	switch url.Scheme {
	case "http", "socks5", "quic":
	case "socks":
		url.Scheme = "socks5"
	default:
		return nil, errors.Errorf("scheme:%s not support\n", url.Scheme)
	}

	return &ProxyNode{
		URL:    *url,
		RawURL: rawurl,
	}, nil
}

func (pn *ProxyNode) EncodeBasicAuth() string {
	var authStr string
	user := pn.URL.User
	if user != nil {
		s := user.String()
		if _, set := user.Password(); !set {
			s += ":"
		}
		authStr = "Basic " + base64.StdEncoding.EncodeToString([]byte(s))
	}
	return authStr
}
