package cmd

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"tinycode/hackerrank"
	"tinycode/leetcode"
	"tinycode/provider"
)

// Flags and parameters
var backend string
var configPath string
var langStr string
var problemId string
var problemSlug string
var contestSlug string
var srcStr string
var doPurchase bool
var debug bool

// State variables
var filters = provider.Filters{}
var config provider.Config

const (
	HackerRankUrl string = "https://www.hackerrank.com/"
	HackerRank           = "hackerrank"
	LeetCodeUrl          = "https://leetcode.com/"
	LeetCode             = "leetcode"
)

type Metadata struct {
	Backend string
	Filters *provider.Filters
}

func GetMetadata(path string) (*Metadata, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(srcFile)
	var parsedFilters provider.Filters
	var parsedBackend string
	for scanner.Scan() {
		ln := scanner.Text()
		re := regexp.MustCompile("([\\w-]+) metadata: ")
		matches := re.FindStringSubmatch(ln)
		if len(matches) != 0 {
			parsedBackend = matches[1]
			log.Printf("metadata found for %s", backend)
			f := provider.ParseFilters(ln)
			parsedFilters = *f
		}
	}
	return &Metadata{
		Backend: parsedBackend,
		Filters: &parsedFilters,
	}, nil
}

func IsConfigCommand(cmd *cobra.Command) bool {
	return strings.HasPrefix(cmd.Use, "login")
}

var client provider.Provider

var rootCmd = &cobra.Command{
	Use:     "tinycode",
	Short:   "Real hackers don't do competitive coding in the browser",
	Example: `  tinycode checkout --difficulty easy --submit /tmp`,
	Version: "0.2.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			log.SetOutput(os.Stderr)
		} else {
			devNull, _ := os.Create(os.DevNull)
			log.SetOutput(devNull)
		}

		if len(args) != 0 {
			srcStr = args[0]
		}

		// if a path is passed as argument, try to set filters and backend flag
		// by looking up the metadata in it
		if srcStr != "" {
			if metadata, err := GetMetadata(srcStr); err == nil {
				if backend == "" && metadata.Backend != "" {
					backend = metadata.Backend
				}
				filters.Update(metadata.Filters)
			}
		}

		// if backend is not specified (and was not overridden by metadata), default
		// is "hackerrank"
		if backend == "" {
			backend = HackerRank
		}

		// instantiate the backend client
		switch backend {
		case LeetCode:
			base, _ := url.Parse(LeetCodeUrl)
			client = leetcode.NewClient(base)
		case HackerRank:
			base, _ := url.Parse(HackerRankUrl)
			hrClient := hackerrank.NewClient(base)
			hrClient.DoPurchase = doPurchase
			client = hrClient
		default:
			return fmt.Errorf("unknown provider: %s (must be hackerrank or leetcode)", backend)
		}

		if IsConfigCommand(cmd) { // cmd is `login` or other configuration subcommand
			return nil
		}

		// read the configuration file and extract+apply the backend config
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("no configuration found: try running: tinycode login -p %s", backend)
		}

		if err := viper.Unmarshal(&config); err != nil {
			return err
		}

		config, in := config.Backend[backend]
		if !in {
			return fmt.Errorf("no authentication token for %s found: try running: tinycode login -p %[1]s", backend)
		}

		if err := client.Configure(config); err != nil {
			return err
		}

		// check if we are signed in, in order to check the validity of our token
		if isSignedIn, err := client.IsSignedIn(); err != nil || !isSignedIn {
			if err != nil {
				log.Printf("error trying to check if signed in: %v", err)
			} else {
				log.Printf("not signed in")
			}
			return fmt.Errorf("current login config for %s invalid: try running: tinycode login -p %[1]s", backend)
		} else {
			log.Printf("valid authentication token found!")
		}

		// if --lang is not given but a path was passed in argument, attempt to
		// recover a --lang value from the file path's extension (if it has an extension)
		if langStr == "" && srcStr != "" {
			spl := strings.Split(args[0], ".")
			if len(spl) > 1 {
				ext := spl[len(spl)-1]
				parsedLang, err := provider.ParseExt(ext)
				if err == nil { // a match
					langStr = parsedLang.String()
				}
			}
		}

		if problemId != "" {
			if err := filters.AddFilter("id", problemId); err != nil {
				return err
			}
		}

		if problemSlug != "" {
			if err := filters.AddFilter("slug", problemSlug); err != nil {
				return err
			}
		}

		if contestSlug == "" && backend == HackerRank {
			contestSlug = "master"
		}

		if contestSlug != "" {
			if err := filters.AddFilter("contest", contestSlug); err != nil {
				return err
			}
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
	rootCmd.PersistentFlags().StringVarP(&backend, "provider", "p", "", "which problem provider to use (leetcode or hackerrank)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debugging output")

	checkoutCmd.Flags().StringVarP(&difficultyStr, "difficulty", "d", "", "limit search to a given difficulty (easy, medium, hard)")
	checkoutCmd.Flags().StringVar(&statusStr, "status", "", "limit search to a given status (todo, attempted, solved)")
	checkoutCmd.Flags().StringVarP(&tagsStr, "tags", "t", "", "limit search to a given list of (comma-separated) tags (leetcode only)")
	checkoutCmd.Flags().StringVar(&problemSlug, "problem", "", "slug of a problem (e.g. two-sum)")
	checkoutCmd.Flags().StringVar(&problemId, "id", "", "id of a problem (e.g. 1)")
	checkoutCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	checkoutCmd.Flags().StringVar(&contestSlug, "contest", "", "contest to which the problem belong (hackerrank only)")
	checkoutCmd.Flags().BoolVarP(&doOpen, "open", "o", false, "whether to open the file")
	checkoutCmd.Flags().StringVar(&trackStr, "track", "", "limit search to a given track (hackerrank only)")
	checkoutCmd.Flags().BoolVarP(&doSubmit, "submit", "s", false, "whether to open the file then submit after closing")
	rootCmd.AddCommand(checkoutCmd)

	submitCmd.Flags().StringVar(&problemSlug, "problem", "", "slug of a problem (e.g. two-sum)")
	submitCmd.Flags().StringVar(&problemId, "id", "", "id of a problem (e.g. 1)")
	submitCmd.Flags().StringVarP(&langStr, "lang", "l", "", "target language of the submission (e.g. cpp)")
	submitCmd.Flags().BoolVar(&doPurchase, "purchase", false, "whether to purchase the last failed testcase (hackerrank only)")
	rootCmd.AddCommand(submitCmd)

	loginCmd.Flags().StringVarP(&csrf, "csrf", "c", "", "Manually set the X-CSRF-Token")
	loginCmd.Flags().StringVarP(&session, "session", "s", "", "Manually set the session token (_hrank_session for hackerrank, LEETCODE_SESSION for leetcode)")
	rootCmd.AddCommand(loginCmd)

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.SetDefault("backend.leetcode.csrf-header", "X-csrftoken")
	viper.SetDefault("backend.hackerrank.csrf-header", "X-CSRF-Token")
	viper.AddConfigPath(configPath)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "tinycode: %s", err)
		os.Exit(1)
	}
}
