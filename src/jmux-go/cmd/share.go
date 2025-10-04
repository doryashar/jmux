package cmd

import (
	"github.com/spf13/cobra"
)

var (
	shareName    string
	sharePrivate bool
	shareInvite  []string
	shareView    bool
	shareRogue   bool
	sharePassword string
	shareSecure   bool
)

// shareCmd represents the share command
var shareCmd = &cobra.Command{
	Use:   "share [session-name]",
	Short: "Share the current tmux session",
	Long: `Share the current tmux session with other users.

Sharing Modes:
  Default: Standard shared session (all users have full control)
  --view:  View-only mode (joining users can only observe, read-only)
  --rogue: Rogue mode (joining users get independent control within same tmux server)

Security Options:
  --secure:   Enable encrypted sessions (requires password)
  --password: Set password for secure sessions

Examples:
  dmux share                              # Share current session publicly
  dmux share tomere                       # Share with name 'tomere'
  dmux share --name mysession             # Share with custom name
  dmux share --view                       # Share in read-only mode
  dmux share --rogue                      # Share in rogue mode (independent sessions)
  dmux share --private --invite user1,user2  # Private session with invites
  dmux share --secure --password mypass   # Secure encrypted session
  dmux share --secure                     # Secure session with config password`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use positional argument if provided, otherwise use flag
		sessionName := shareName
		if len(args) > 0 {
			sessionName = args[0]
		}
		
		// Validate mutually exclusive flags
		if shareView && shareRogue {
			cmd.Printf("Error: --view and --rogue flags are mutually exclusive\n")
			return
		}

		// Validate security options
		if shareSecure && sharePassword == "" && cfg.Security.GlobalPassword == "" {
			cmd.Printf("Error: --secure requires a password (use --password or configure global password)\n")
			return
		}

		// Determine sharing mode
		var shareMode string
		switch {
		case shareView:
			shareMode = "view"
		case shareRogue:
			shareMode = "rogue"
		default:
			shareMode = "pair"
		}

		// Configure security for this session if requested
		if shareSecure {
			// Create a copy of the config with security enabled
			secureConfig := *cfg.Security
			secureConfig.Enabled = true
			
			// Set session-specific password if provided
			if sharePassword != "" {
				if secureConfig.SessionPasswords == nil {
					secureConfig.SessionPasswords = make(map[string]string)
				}
				secureConfig.SessionPasswords[sessionName] = sharePassword
			}
			
			// Update the config temporarily for this session
			originalConfig := cfg.Security
			cfg.Security = &secureConfig
			defer func() {
				cfg.Security = originalConfig
			}()
		}

		err := sessMgr.StartShare(sessionName, sharePrivate, shareInvite, shareMode)
		if err != nil {
			cmd.Printf("Error starting share: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(shareCmd)

	shareCmd.Flags().StringVar(&shareName, "name", "", "Custom session name")
	shareCmd.Flags().BoolVar(&sharePrivate, "private", false, "Create private session")
	shareCmd.Flags().StringSliceVar(&shareInvite, "invite", []string{}, "Users to invite (comma-separated)")
	shareCmd.Flags().BoolVar(&shareView, "view", false, "Share in view-only mode (read-only for joining users)")
	shareCmd.Flags().BoolVar(&shareRogue, "rogue", false, "Share in rogue mode (independent control for joining users)")
	shareCmd.Flags().BoolVar(&shareSecure, "secure", false, "Enable encrypted session (requires password)")
	shareCmd.Flags().StringVar(&sharePassword, "password", "", "Password for secure session")
}