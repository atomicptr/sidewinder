package cli

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/atomicptr/sidewinder/pkg/config"
	"github.com/atomicptr/sidewinder/pkg/service"
	"github.com/spf13/cobra"
)

var configFile string = ""
var dataDir string = ""
var tickRate time.Duration

const defaultTickRate = 30 * time.Minute

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run the service",
	Run: func(cmd *cobra.Command, args []string) {
		if configFile == "" {
			configFile = os.Getenv("SIDEWINDER_CONFIG_FILE")
		}

		if configFile == "" {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}

			configFile = filepath.Join(cwd, "sidewinder.toml")
		}

		if dataDir == "" {
			dataDir = os.Getenv("SIDEWINDER_CONFIG_FILE")
		}

		if dataDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}

			dataDir = filepath.Join(cwd, "data")
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			log.Fatal(err)
		}

		err := os.MkdirAll(dataDir, 0o755)
		if err != nil {
			log.Fatal(err)
		}

		f, err := os.Open(configFile)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				log.Fatal(err)
			}
		}()

		config, err := config.Read(f)

		if config.TickRate == 0 {
			tr := os.Getenv("SIDEWINDER_TICK_RATE")
			if tr != "" {
				t, err := time.ParseDuration(tr)
				if err != nil {
					log.Fatal(err)
				}
				config.TickRate = t
			}
		}

		if tickRate != defaultTickRate && tickRate != 0 {
			config.TickRate = tickRate
		}

		if config.TickRate == 0 {
			config.TickRate = defaultTickRate
		}

		err = service.Run(config, dataDir)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	runCommand.PersistentFlags().StringVarP(&configFile, "config-file", "", "", "set path to config file, reads env var: SIDEWINDER_CONFIG_FILE")
	runCommand.PersistentFlags().StringVarP(&dataDir, "data-dir", "", "", "set path to data dir, reads env var: SIDEWINDER_DATA_DIR")
	runCommand.PersistentFlags().DurationVarP(&tickRate, "tick-rate", "", defaultTickRate, "set tick rate (time between RSS feed pulls), reads env var: SIDEWINDER_TICK_RATE")
}
