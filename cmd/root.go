package cmd

import (
	"fmt"

	"github.com/snana7mi/conchtalk-dlc/daemon"
	"github.com/spf13/cobra"
)

var (
	token   string
	server  string
	version string
)

func SetVersion(v string) {
	version = v
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("conchtalk-dlc", version)
	},
}

var rootCmd = &cobra.Command{
	Use:   "conchtalk-dlc",
	Short: "ConchTalk DLC — remote tool executor for ConchTalk relay mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			return fmt.Errorf("--token is required")
		}
		return daemon.Run(token, server, version)
	},
}

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "Relay authentication token (required)")
	rootCmd.Flags().StringVar(&server, "server", "wss://api.conch-talk.com/relay", "Relay server WebSocket URL")
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
