package dns

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}

type config struct {
	Sources          []string
	GeoIPSrc         string
	GeoIPName        string
	DataDir          string
	Blocklist        []string
	Whitelist        []string
	Bind             string
	API              string
	Nullroute        string
	Nullroutev6      string
	Nameservers      []string
	CHNameservers    []string
	ISPNameservers   []string
	Interval         duration
	Timeout          duration
	SessionTimeout   duration
	Expire           duration
	Maxcount         int
	QuestionCacheCap int
	TTL              uint32
	FakeInterval     duration
	FakeIps          []string
}

var defaultConfig = `# list of sources to pull blocklists from, stores them in sourcedir
sources = [
	"http://mirror1.malwaredomains.com/files/justdomains",
	"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	"http://sysctl.org/cameleon/hosts",
	"https://zeustracker.abuse.ch/blocklist.php?download=domainblocklist",
	"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
	"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
	"http://hosts-file.net/ad_servers.txt",
	"https://raw.githubusercontent.com/quidsup/notrack/master/trackers.txt"
]

# source of GeoIP database
geoipSrc = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"

# local file name of GeoIP database
geoipName = "GeoLite2-City.mmdb"

# locations to store blocklist files and GeoIP database
datadir = "./data"

# manual blocklist entries
blocklist = []

# manual whitelist entries
whitelist = [
	"126.com",
	"163.com",
	"getsentry.com",
	"www.getsentry.com"
]

# address to bind to for the DNS server
bind = "0.0.0.0:53"

# address to bind to for the API server
api = "127.0.0.1:8080"

# ipv4 address to forward blocked queries to
nullroute = "0.0.0.0"

# ipv6 address to forward blocked queries to
nullroutev6 = "0:0:0:0:0:0:0:0"

# nameservers(not in China) to forward queries to
nameservers = [
	"208.67.222.222:443", # opendns
	"208.67.222.222:5353", # opendns
	"208.67.220.220:443", # opendns
	"208.67.220.123:443",
	"80.90.43.162:5678",
	"113.20.6.2:443",
	"113.20.8.17:443",
	"95.141.34.162:5678",
	"77.66.84.233:443",
	"176.56.237.171:443",
	"142.4.204.111:443",
	"178.216.201.222:2053",
	"8.8.8.8:53", # google
	"8.8.4.4:53", # google
	"208.67.222.222:53", # opendns
	"208.67.220.220:53", # opendns
	"74.82.42.42:53" # he
]

# nameservers in China
chnameservers = [
	"1.2.4.8:53",
	"210.2.4.8:53",
	"114.114.114.114:53",
	"114.114.115.115:53",
	"182.254.116.116:53",
	"182.254.118.118:53",
	"223.5.5.5:53",
	"223.6.6.6:53"
]

# nameservers for ISP or enterprise network, maybe null
ispnameservers = [
]

# concurrency interval for lookups
interval = "100ms"

# timeout for one dns lookup message
timeout = "800ms"

# timeout for one dns lookup session(one message for on target)
sessiontimeout = "2s"

# cache entry lifespan
expire = "3600s"

# cache capacity, 0 for infinite
maxcount = 0

# question cache capacity, 0 for infinite but not recommended (this is used for storing logs)
questioncachecap = 5000

# interval for fake ip discovery
fakeInterval = "30s"

# fake ip for cold boot, please change it for your networks
# you only need a few common fake ip, ghost will discover other fake ips regularly
fakeIPs = [
	"93.46.8.89",
	"8.7.198.45",
	"203.98.7.65",
	"46.82.174.68",
	"78.16.49.15",
	"59.24.3.173",
	"37.61.54.158"
]
`

// Config is the global configuration
var gConfig config

// LoadConfig loads the given config file
func LoadConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := generateConfig(path); err != nil {
			return err
		}
	}

	if _, err := toml.Decode(defaultConfig, &gConfig); err != nil {
		return errors.Wrap(err, "failed to load default config")
	}

	if _, err := toml.DecodeFile(path, &gConfig); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	gQuestionCache.Maxcount = gConfig.QuestionCacheCap

	return nil
}

func generateConfig(path string) error {
	output, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer output.Close()

	r := strings.NewReader(defaultConfig)
	if _, err := io.Copy(output, r); err != nil {
		return errors.Wrap(err, "failed to copy default config")
	}

	return nil
}
