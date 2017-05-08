package tun

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type ProxyNode struct {
	URL    *url.URL
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
	case "http":
	default:
		return nil, errors.Errorf("Scheme:%s not support\n", url.Scheme)
	}

	return &ProxyNode{
		URL:    url,
		RawURL: rawurl,
	}, nil
}
