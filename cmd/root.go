package cmd

import (
	"fmt"
	"github.com/brokad/tinycode/hackerrank"
	"github.com/brokad/tinycode/leetcode"
	"github.com/brokad/tinycode/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
)

var backend string
var configPath string

// Filters
var langStr string
var lang provider.Lang

var problemId string
var problemSlug string
var contestSlug string

var filters = provider.Filters{}

var baseStr string
var pwd string
var baseUrl *url.URL
var srcStr string

var doPurchase bool

var debug bool

var config provider.Config

var client provider.Provider

func printCookieReadmeAndExit(provider string) {
	var instructions string
	switch provider {
	case "leetcode":
		instructions = fmt.Sprintf(`
  (1) Head over to https://leetcode.com/accounts/login/ and login to leetcode
      using Firefox or Chrome/Chromium`)
	default:
		log.Panicf("unknown provider: %s", provider)
	}
	fmt.Fprintf(os.Stderr, `
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Could not find a valid authentication token for %[1]s.

tinycode needs a current API token in order to query %[1]s's API. Please follow 
these instructions to obtain one:
%[2]s
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

`, provider, instructions)
	log.Fatalf("could not authenticate to %s", provider)
}

var rootCmd = &cobra.Command{
	Use:   "tinycode",
	Short: "Crunch competitive coding questions in your favorite IDE, not in the browser",
	Example: `  tinycode checkout --difficulty easy --open problem.cpp
  tinycode submit problem.cpp`,
	Version: "0.1.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			log.SetOutput(os.Stderr)
		} else {
			devNull, _ := os.Create(os.DevNull)
			log.SetOutput(devNull)
		}

		if err := viper.ReadInConfig(); err != nil {
			return err
		}

		if err := viper.Unmarshal(&config); err != nil {
			return err
		}

		backendConfig, in := config.Backend[backend]
		if !in {
			printCookieReadmeAndExit(backend)
		}

		switch backend {
		case "leetcode":
			baseStr = "https://www.leetcode.com"
			base, err := url.Parse(baseStr)
			if err != nil {
				return err
			}
			client, err = leetcode.NewClient(backendConfig, base)
			if err != nil {
				return err
			}
		case "hackerrank":
			baseStr = "https://www.hackerrank.com"
			base, err := url.Parse(baseStr)
			if err != nil {
				return err
			}
			hrClient, err := hackerrank.NewClient(backendConfig, base)
			if err != nil {
				return err
			}
			hrClient.DoPurchase = doPurchase
			client = hrClient
		default:
			return fmt.Errorf("unknown provider: %s", backend)
		}

		if isSignedIn, err := client.IsSignedIn(); err != nil || !isSignedIn {
			if err != nil {
				log.Printf("error trying to check if user is signed in: %s", err)
			}
			if !isSignedIn {
				log.Printf("user is not signed in")
			}
			printCookieReadmeAndExit(backend)
		} else {
			log.Printf("valid authentication token found")
		}

		if langStr == "" && len(args) == 1 {
			spl := strings.Split(args[0], ".")
			if len(spl) > 1 {
				ext := spl[len(spl)-1]
				parsedLang, err := provider.ParseExt(ext)
				langStr = parsedLang.String()
				if err != nil {
					return err
				}
			}
		}

		if langStr != "" {
			if parsedLang, err := provider.ParseLang(langStr); err != nil {
				return err
			} else {
				lang = *parsedLang
			}

			localLangStr, err := client.LocalizeLanguage(lang)
			if err != nil {
				return err
			} else {
				langStr = localLangStr
			}

			filters.AddFilter("lang", langStr)
		} else {
			return fmt.Errorf("a --lang must be provided (e.g. rust)")
		}

		if len(args) != 0 {
			srcStr = args[0]
		}

		if problemId != "" {
			filters.AddFilter("id", problemId)
		}

		if problemSlug != "" {
			filters.AddFilter("slug", problemSlug)
		}

		if contestSlug == "" && backend == "hackerrank" {
			contestSlug = "master"
		}

		if contestSlug != "" {
			filters.AddFilter("contest", contestSlug)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	usr, _ := user.Current()
	home := usr.HomeDir
	configPathDefault := path.Join(home, ".config/tinycode")

	rootCmd.PersistentFlags().StringVar(&configPath, "config", configPathDefault, "the path to the configuration directory")
	rootCmd.MarkFlagDirname("config")
	rootCmd.PersistentFlags().StringVar(&backend, "backend", "leetcode", "which problem set provider to use (leetcode or hackerrank)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debugging output")

	checkoutCmd.Flags().StringVarP(&difficultyStr, "difficulty", "d", "", "limit search to a given difficulty (easy, medium, hard)")
	checkoutCmd.Flags().StringVar(&statusStr, "status", "", "limit search to a given status (todo, attempted, solved)")
	checkoutCmd.Flags().StringVarP(&tagsStr, "tags", "t", "", "limit search to a given list of (comma-separated) tags")
	checkoutCmd.Flags().StringVarP(&problemSlug, "problem", "p", "", "slug of a problem (e.g. two-sum)")
	checkoutCmd.Flags().StringVarP(&problemId, "id", "i", "", "id of a problem (e.g. 1)")
	checkoutCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	checkoutCmd.Flags().StringVarP(&contestSlug, "contest", "c",  "","contest to which the problem belong (hackerrank only)")
	checkoutCmd.Flags().BoolVarP(&doOpen, "open", "o", false, "whether to open the file")
	rootCmd.AddCommand(checkoutCmd)

	submitCmd.Flags().StringVarP(&problemSlug, "problem", "p", "", "slug of a problem (e.g. two-sum)")
	submitCmd.Flags().StringVarP(&problemId, "id", "i", "", "id of a problem (e.g. 1)")
	submitCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	submitCmd.Flags().BoolVar(&doPurchase, "purchase", false, "whether to purchase the last failed testcase (hackerrank only)")
	rootCmd.AddCommand(submitCmd)

	pwd, _ = os.Getwd()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.SetDefault("backend.leetcode.csrf-header", "x-csrftoken")
	viper.SetDefault("backend.hackerrank.csrf-header", "X-CSRF-Token")
	viper.AddConfigPath(configPath)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
