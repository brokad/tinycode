package cmd

import (
	"errors"
	"fmt"
	"github.com/brokad/tinycode/leetcode"
	"github.com/spf13/cobra"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
)

var backend string
var configPath string
var langStr string

var problemId string
var slug string
var lang leetcode.LangSlug

var baseStr string
var pwd string
var baseUrl *url.URL
var srcStr string

var client *leetcode.Client

func printCookieReadmeAndExit(provider string, path string) {
	var instructions string
	switch provider {
	case "leetcode":
		instructions = fmt.Sprintf(`
  (1) Head over to https://leetcode.com/accounts/login/ and login to leetcode

  (2) Open the development console (e.g. F12 for Firefox)

  (3) Open the network inspection tab (e.g. Network for Firefox)

  (4) Look for an API call to 'graphql', refreshing the page if necessary
  
  (5) In the request header section of the inspector window, look for 'Cookie: '
      header and copy/paste the _value_ to the file:

         %s

Be careful to include the whole cookie as some browsers (e.g. Firefox) truncate
their output in the inspector window. Follow your browser's documentation (e.g. 
for Firefox click on the 'Raw' toggle).`, path)
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
	Use: "tinycode",
	Short:   "Crunch LeetCode questions in your favorite IDE, not in the browser",
	Example: `  tinycode checkout --difficulty easy --open problem.cpp
  tinycode submit problem.cpp`,
	Version: "0.1.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch backend {
		case "leetcode":
			baseStr = "https://leetcode.com"
		default:
			return fmt.Errorf("unknown provider: %s", backend)
		}

		base, err := url.Parse(baseStr)
		if err != nil {
			return err
		} else {
			baseUrl = base
		}

		cookieJarStr := path.Join(configPath, fmt.Sprintf("%s.cookies", backend))
		cookieFile, err := os.Open(cookieJarStr)
		if errors.Is(err, os.ErrNotExist) {
			printCookieReadmeAndExit(backend, cookieJarStr)
		} else if err != nil {
			return err
		}

		client, err = leetcode.NewClient(cookieFile, base)
		if err != nil {
			return err
		}

		if isSignedIn, err := client.IsSignedIn(); err != nil || !isSignedIn {
			if err != nil {
				log.Printf("error trying to check if user is signed in: %s", err)
			}
			if !isSignedIn {
				log.Printf("user is not signed in")
			}
			printCookieReadmeAndExit(backend, cookieJarStr)
		} else {
			log.Printf("valid authentication token found")
		}

		if langStr == "" && len(args) == 1 {
			spl := strings.Split(args[0], ".")
			if len(spl) > 1 {
				ext := spl[len(spl)-1]
				langSlug, err := leetcode.NewLangFromExt(ext)
				langStr = string(langSlug)
				if err != nil {
					return err
				}
			}
		}

		if langStr != "" {
			lang = leetcode.LangSlug(langStr)
		} else {
			return fmt.Errorf("a --lang must be provided (e.g. rust)")
		}

		if len(args) != 0 {
			srcStr = args[0]
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

	rootCmd.Flags().StringVar(&configPath, "config", configPathDefault, "the path to the configuration directory")
	rootCmd.MarkFlagDirname("config")
	rootCmd.Flags().StringVar(&backend, "backend", "leetcode", "which problem set provider to use (only leetcode allowed)")

	checkoutCmd.Flags().StringVarP(&difficultyStr, "difficulty", "d", "", "limit search to a given difficulty (easy, medium, hard)")
	checkoutCmd.Flags().StringVar(&statusStr, "status", "", "limit search to a given status (todo, attempted, solved)")
	checkoutCmd.Flags().StringVarP(&tagsStr, "tags", "t", "", "limit search to a given list of (comma-separated) tags")
	checkoutCmd.Flags().StringVarP(&slug, "problem-slug", "p", "", "slug of a problem (e.g. two-sum)")
	checkoutCmd.Flags().StringVarP(&problemId, "problem-id", "i", "", "id of a problem (e.g. 1)")
	checkoutCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	checkoutCmd.Flags().BoolVarP(&doOpen, "open", "o", false, "whether to open the file")
	rootCmd.AddCommand(checkoutCmd)

	submitCmd.Flags().StringVarP(&slug, "problem-slug", "p", "", "slug of a problem (e.g. two-sum)")
	submitCmd.Flags().StringVarP(&problemId, "problem-id", "i", "", "id of a problem (e.g. 1)")
	submitCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	rootCmd.AddCommand(submitCmd)

	pwd, _ = os.Getwd()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
