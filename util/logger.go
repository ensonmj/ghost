package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// LoggerInit Initializes the logger
func LoggerInit(logFile string) (*os.File, error) {
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		if _, err := os.Create(logFile); err != nil {
			return nil, fmt.Errorf("failed to create log file: %s", err)
		}
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %s", err)
	}

	logWriter := io.MultiWriter(os.Stdout, file)

	log.SetOutput(logWriter)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	return file, nil
}

func FlushDir(dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dirPath, name))
		if err != nil {
			return err
		}
	}
	return nil
}
