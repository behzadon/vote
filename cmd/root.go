package cmd

import (
	"fmt"
	"os"

	"github.com/behzadon/vote/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
	rootCmd = &cobra.Command{
		Use:   "vote",
		Short: "Interactive polling platform",
		Long: `A massively scalable, interactive polling platform that provides 
a vertical feed of polls for mobile and web applications.`,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
}

func GetConfig() *config.Config {
	return cfg
}
