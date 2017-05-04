package tun

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetHttpProxyHandler(t *testing.T) {
	// http server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "success")
	}))
	defer srv.Close()

	// http proxy server
	proxySrv := httptest.NewServer(GetHttpProxyHandler(false))
	defer proxySrv.Close()

	// http client
	proxyUrl, _ := url.Parse(proxySrv.URL)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyURL(proxyUrl),
		},
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Error(err)
	}
	txt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Error(err)
	}
	if string(txt) != "success" {
		t.Errorf("expect success, but got %s\n", txt)
	}
}
