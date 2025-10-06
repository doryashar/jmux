package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"jmux/internal/jcat"
)

var secureFlag bool

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
		
		// Start the appropriate server type
		if secureFlag && cfg != nil && cfg.Security.Enabled {
			secureServer := jcat.NewSecureServer(fmt.Sprintf(":%d", port), setSizeScript, cfg.Security)
			if err := secureServer.Start(); err != nil {
				fmt.Printf("secure jcat server error: %v\n", err)
			}
		} else {
			server := jcat.NewServer(fmt.Sprintf(":%d", port), setSizeScript)
			if err := server.Start(); err != nil {
				fmt.Printf("jcat server error: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(internalJcatServerCmd)
	internalJcatServerCmd.Flags().BoolVar(&secureFlag, "secure", false, "Start secure server")
}