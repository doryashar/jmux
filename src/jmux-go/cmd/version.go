package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"jmux/internal/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for jmux-go including build details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			fmt.Println(version.GetFullVersion())
		} else {
			fmt.Println(version.GetVersion())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
}