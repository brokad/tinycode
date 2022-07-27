package cmd

import (
	"fmt"
	"github.com/brokad/tinycode/hackerrank"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var csrf string
var session string

var loginCmd = &cobra.Command{
	Use:     "login [-c CSRF] [-s TOKEN] [-p hackerrank | -p leetcode]",
	Short:   "configure authentication for problem set providers",
	Example: `  tinycode login -p hackerrank`,
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
			return err
		}

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				path := filepath.Dir(viper.ConfigFileUsed())
				if err := os.MkdirAll(path, os.ModePerm); err != nil {
					return err
				}
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
		} else {
			return fmt.Errorf("no --csrf or --session: not doing anything")
		}

		return nil
	},
}
