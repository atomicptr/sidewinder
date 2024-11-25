package cli

import "github.com/spf13/cobra"

var rootCommand = &cobra.Command{
	Use:   "sidewinder",
	Short: "Sidewinder is a simple tool that reads RSS feeds and posts them to chat channels via webhooks",
}

func init() {
	rootCommand.AddCommand(runCommand)
}
