package cmd

import (
	"fmt"
	"github.com/brokad/tinycode/hackerrank"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var csrf string
var session string

var loginCmd = &cobra.Command{
	Use:     "login [-c CSRF] [-s TOKEN] [-p hackerrank | -p leetcode]",
	Short:   "configure authentication for problem set providers",
	Example: `  tinycode login -p hackerrank`,
	Args:    cobra.ExactArgs(0),
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

		switch c := client.(type) {
		case *hackerrank.Client:
			newCsrf, newSession, err := c.GetLogIn()
			if err != nil {
				return err
			} else {
				csrf = newCsrf
				session = newSession
			}
		default:
			return fmt.Errorf("unknown provider: %s", backend)
		}

		var mutate = false

		if csrf != "" {
			viper.Set(fmt.Sprintf("backend.%s.csrf", backend), csrf)
			mutate = true
		}

		if session != "" {
			viper.Set(fmt.Sprintf("backend.%s.session", backend), session)
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
