package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var csrf string
var auth string

var loginCmd = &cobra.Command {
	Use: "login [--csrf CSRF] [--session TOKEN] PROVIDER",
	Short: "configure authentication for problem set providers",
	Example: `  tinycode login hackerrank --csrf="abcdabcd" --auth="abcdabcd"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				if err := viper.SafeWriteConfig(); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		backend = args[0]

		var mutate = false

		if csrf != "" {
			viper.Set(fmt.Sprintf("backend.%s.csrf", backend), csrf)
			mutate = true
		}

		if auth != "" {
			viper.Set(fmt.Sprintf("backend.%s.auth", backend), auth)
			mutate = true
		}

		if mutate {
			if err := viper.WriteConfig(); err != nil {
				return err
			}
		}

		return nil
	},
}