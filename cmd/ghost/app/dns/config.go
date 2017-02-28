package dns

import (
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type config struct {
	Sources          []string
	SourceDir        string
	Blocklist        []string
	Whitelist        []string
	Bind             string
	API              string
	Nullroute        string
	Nullroutev6      string
	Nameservers      []string
	Interval         int
	Timeout          int
	Expire           int
	Maxcount         int
	QuestionCacheCap int
	TTL              uint32
	FakeInterval     int
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

# locations to store blocklist files
sourcedir = "./sources"

# manual blocklist entries
blocklist = []

# manual whitelist entries
whitelist = [
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

# nameservers to forward queries to
nameservers = [
	"1.2.4.8:53",
	"8.8.8.8:53",
	"8.8.4.4:53"
]

# concurrency interval for lookups in miliseconds
interval = 200

# query timeout for dns lookups in seconds
timeout = 60

# cache entry lifespan in seconds
expire = 600

# cache capacity, 0 for infinite
maxcount = 0

# question cache capacity, 0 for infinite but not recommended (this is used for storing logs)
questioncachecap = 5000

# interval for fake ip lookups in second
fakeInterval = 30
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
