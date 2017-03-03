package app

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ensonmj/ghost/cmd/ghost/app/dns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fDNSConfigPath  string
	fDNSForceUpdate bool
)

var DNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS",
	RunE:  dnsMain,
}

func init() {
	flags := DNSCmd.Flags()
	flags.StringVar(&fDNSConfigPath, "config", "dns.toml", "location of the config file, if not found it will be generated (default dns.toml)")
	flags.BoolVar(&fDNSForceUpdate, "update", false, "force an update of the blocklist file")
}

func dnsMain(cmd *cobra.Command, args []string) error {
	if err := dns.LoadConfig(fDNSConfigPath); err != nil {
		return err
	}

	go dns.UpdateFakeIP("114.114.114.114:53")
	go dns.StartAPIServer(viper.GetBool("debug"))
	dns.NewServer(5*time.Second, 5*time.Second).AsyRun()

	// need to resolve domains for file downloads
	// we should start our dns server firstly
	if err := dns.LoadData(fDNSForceUpdate); err != nil {
		return err
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("signal received")

	return nil
}
