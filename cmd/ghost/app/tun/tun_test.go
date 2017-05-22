package tun

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	ret := m.Run()
	teardown()
	os.Exit(ret)
}

func setup() {
	flag.Parse()
	if testing.Verbose() {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func teardown() {}

func setupSrvAndClient(tr *http.Transport) error {
	// http server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "success")
	}))
	defer srv.Close()

	// http client with proxy
	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err != nil {
		return err
	}
	txt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if string(txt) != "success" {
		errors.New(fmt.Sprintf("expect success, but got %s\n", txt))
	}
	return nil
}
