package cmd

import (
	"github.com/spf13/cobra"
)

var (
	token  string
	server string
)

var rootCmd = &cobra.Command{
	Use:   "conchtalk-dlc",
	Short: "ConchTalk DLC — remote tool executor for ConchTalk relay mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			return cmd.Help()
		}
		// Will be implemented in Task 3
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "Relay authentication token (required)")
	rootCmd.Flags().StringVar(&server, "server", "wss://api.conch-talk.com/relay", "Relay server WebSocket URL")
}

func Execute() error {
	return rootCmd.Execute()
}
