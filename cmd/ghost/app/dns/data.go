package dns

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// LoadData loads the BlockCache
func LoadData(forceupdate bool) error {
	log.Printf("loading blocked domains from %s\n", gConfig.SourceDir)

	if _, err := os.Stat(gConfig.SourceDir); os.IsNotExist(err) {
		if err := os.Mkdir(gConfig.SourceDir, 0700); err != nil {
			return errors.Wrapf(err, "failed to create source directory: %s",
				gConfig.SourceDir)
		}
	}

	whitelist := make(map[string]bool)
	for _, entry := range gConfig.Whitelist {
		whitelist[entry] = true
	}

	for _, entry := range gConfig.Blocklist {
		gBlockCache.Set(entry, true)
	}

	for _, uri := range gConfig.Sources {
		u, _ := url.Parse(uri)
		fileName := fmt.Sprintf("%s%s", u.Host, strings.Replace(u.Path, "/", "-", -1))
		path := filepath.Join(gConfig.SourceDir, fileName)
		if _, err := os.Stat(path); os.IsNotExist(err) || forceupdate {
			log.Printf("fetching source %s\n", uri)
			if err := downloadFile(uri, path); err != nil {
				log.Printf("failed to fetch source: %s\n", err)
			}
		}

		if err := parseHostFile(path, whitelist); err != nil {
			return err
		}
	}

	log.Printf("%d domains loaded from sources\n", gBlockCache.Length())

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
