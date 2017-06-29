package app

import (
	"crypto/tls"
	"log"
	"os"
	"sync"

	"github.com/ensonmj/ghost/cmd/ghost/app/tun"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	fChainNodes []string
	fLocalNodes []string
	fCertFile   string
	fKeyFile    string
)

var TunCmd = &cobra.Command{
	Use:   "tun",
	Short: "tunnel",
	RunE:  tunMain,
}

func init() {
	flags := TunCmd.Flags()
	flags.StringSliceVarP(&fChainNodes, "Forward", "F", nil,
		"forward address, can make a forward chain")
	flags.StringSliceVarP(&fLocalNodes, "Listen", "L", []string{"127.0.0.1:8088"},
		"listen address, can listen on multiple ports")
	flags.StringVar(&fCertFile, "cert", "cert.crt", "certificate file for TLS")
	flags.StringVar(&fKeyFile, "key", "key.pem", "key file for TLS")
}

func tunMain(cmd *cobra.Command, args []string) error {
	// cert
	if _, err := os.Stat(fCertFile); os.IsNotExist(err) {
		if err := tun.CreateCertificate(true, fCertFile, fKeyFile); err != nil {
			return err
		}
	}
	cert, err := tls.LoadX509KeyPair(fCertFile, fKeyFile)
	if err != nil {
		return errors.Wrap(err, "failed to load cert")
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	// chain
	pc, err := tun.ParseProxyChain(fChainNodes...)
	if err != nil {
		return err
	}

	// listen
	var wg sync.WaitGroup
	for _, strNode := range fLocalNodes {
		pn, err := tun.ParseProxyNode(strNode)
		if err != nil {
			log.Println(err)
			continue
		}

		wg.Add(1)
		go func(pn *tun.ProxyNode) {
			defer wg.Done()
			log.Printf("proxy listen and serve err: %+v\n",
				tun.NewProxyServer(pn, pc, tlsConfig).ListenAndServe())
		}(pn)
	}
	wg.Wait()

	return nil
}
