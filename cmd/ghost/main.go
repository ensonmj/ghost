package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ensonmj/ghost/cmd/ghost/app"
	"github.com/ensonmj/ghost/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fDebug    bool
	fLogDir   string
	fFlushLog bool
)

func main() {
	viper.SetEnvPrefix("GHOST")
	viper.AutomaticEnv()

	var logFile *os.File
	cmd := &cobra.Command{
		Use:   "ghost",
		Short: "Ghost is an app which can cache dns results now.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !fDebug {
				log.SetFlags(log.LstdFlags | log.Lshortfile)
				return nil
			}
			if _, err := os.Stat(fLogDir); os.IsNotExist(err) {
				if err = os.Mkdir(fLogDir, os.ModePerm); err != nil {
					return errors.Wrapf(err, "failed to create log directory: %s", fLogDir)
				}
			} else if fFlushLog {
				if err := util.FlushDir(fLogDir); err != nil {
					return errors.Wrapf(err, "failed to flush log directory: %s", fLogDir)
				}
			}
			var err error
			logPath := filepath.Join(fLogDir, "ghost.log")
			logFile, err = util.LoggerInit(logPath)
			return errors.Wrapf(err, "failed to create log file: %s", logPath)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if fDebug && logFile != nil {
				return logFile.Close()
			}
			return nil
		},
	}
	cmd.AddCommand(app.VerCmd)
	cmd.AddCommand(app.DNSCmd)
	cmd.AddCommand(app.TunCmd)
	pflags := cmd.PersistentFlags()
	pflags.BoolVarP(&fDebug, "debug", "D", false, "Open log for debug")
	pflags.StringVarP(&fLogDir, "logDir", "L", "./log", "dir for store log file")
	pflags.BoolVar(&fFlushLog, "flushLog", false, "delete old log file before excute")
	viper.BindPFlag("debug", pflags.Lookup("debug"))

	if err := cmd.Execute(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(-1)
	}
}
