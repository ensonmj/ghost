package dns

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
)

var gGeoIP *geoip2.Reader

// LoadData loads the BlockCache
func LoadData(forceupdate bool) error {
	var err error
	if _, err = os.Stat(gConfig.DataDir); os.IsNotExist(err) {
		if err = os.Mkdir(gConfig.DataDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "failed to create source directory: %s",
				gConfig.DataDir)
		}
	}

	whitelist := make(map[string]bool)
	for _, entry := range gConfig.Whitelist {
		whitelist[entry] = true
	}

	for _, entry := range gConfig.Blocklist {
		gBlockCache.Set(entry, true)
	}

	log.Printf("loading blocked domains from %s\n", gConfig.DataDir)
	for _, uri := range gConfig.Sources {
		u, _ := url.Parse(uri)
		fileName := fmt.Sprintf("%s%s", u.Host, strings.Replace(u.Path, "/", "-", -1))
		path := filepath.Join(gConfig.DataDir, fileName)
		if _, err = os.Stat(path); os.IsNotExist(err) || forceupdate {
			log.Printf("fetching source %s\n", uri)
			if err = downloadFile(uri, path); err != nil {
				return err
			}
		}

		if err = parseHostFile(path, whitelist); err != nil {
			return err
		}
	}
	log.Printf("%d domains loaded from sources\n", gBlockCache.Length())

	log.Println("loading GeoIP database")
	dbPath := filepath.Join(gConfig.DataDir, gConfig.GeoIPName)
	if _, err = os.Stat(dbPath); os.IsNotExist(err) || forceupdate {
		gzPath := filepath.Join(gConfig.DataDir, filepath.Base(gConfig.GeoIPSrc))
		if _, err = os.Stat(gzPath); os.IsNotExist(err) || forceupdate {
			if err = downloadFile(gConfig.GeoIPSrc, gzPath); err != nil {
				return err
			}
		}

		f, err := os.Open(gzPath)
		if err != nil {
			return errors.Wrap(err, "failed to open compressed GeoIP database")
		}
		defer f.Close()

		gz, err := gzip.NewReader(f)
		if err != nil {
			return errors.Wrap(err, "failed to decompress GeoIP database")
		}
		defer gz.Close()

		path := filepath.Join(gConfig.DataDir, gConfig.GeoIPName)
		fw, err := os.Create(path)
		if err != nil {
			return errors.Wrap(err, "failed to create GeoIP file")
		}
		defer fw.Close()

		_, err = io.Copy(fw, gz)
		if err != nil {
			return errors.Wrap(err, "failed to write GeoIP file")
		}
	}

	if gGeoIP, err = geoip2.Open(dbPath); err != nil {
		return errors.Wrapf(err, "failed to open geoip database:%s\n", dbPath)
	}
	log.Println("finish to load GeoIP database")

	return nil
}

func downloadFile(uri string, path string) error {
	output, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", path)
	}
	defer output.Close()

	response, err := http.Get(uri)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return errors.Wrapf(err, "failed to download source: %s", uri)
	}

	if _, err := io.Copy(output, response.Body); err != nil {
		return errors.Wrap(err, "failed to copy output")
	}

	return nil
}

func parseHostFile(path string, whitelist map[string]bool) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open file: %s", path)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)

			if len(fields) > 1 && !strings.HasPrefix(fields[1], "#") {
				line = fields[1]
			} else {
				line = fields[0]
			}

			if !gBlockCache.Exists(line) && !whitelist[line] {
				gBlockCache.Set(line, true)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrapf(err, "failed to scan hostfile: %s", path)
	}

	return nil
}
