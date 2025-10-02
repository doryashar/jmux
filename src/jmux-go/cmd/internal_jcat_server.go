package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"jmux/internal/jcat"
)

// internalJcatServerCmd is a hidden command to run jcat server inside tmux
var internalJcatServerCmd = &cobra.Command{
	Use:    "_internal_jcat_server [port] [setsize-script]",
	Short:  "Internal command to run jcat server",
	Long:   `Internal command used to run jcat server inside tmux sessions. Not for direct use.`,
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid port: %v\n", err)
			return
		}
		
		setSizeScript := args[1]
		
		// Start the jcat server
		server := jcat.NewServer(fmt.Sprintf(":%d", port), setSizeScript)
		if err := server.Start(); err != nil {
			fmt.Printf("jcat server error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(internalJcatServerCmd)
}