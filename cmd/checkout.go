package cmd

import (
	"errors"
	"fmt"
	"github.com/brokad/tinycode/provider"
	"github.com/iancoleman/strcase"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

var difficultyStr string
var statusStr string
var tagsStr string
var trackStr string

var doOpen bool
var doSubmit bool

func toFileIfNotExists(path string, content string) error {
	_, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		log.Printf("file already exists: %s, not writing anything", path)
	} else {
		log.Printf("writing to file %s", path)
		output, err := os.Create(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s", content)
		output.Close()
	}
	fmt.Fprintf(os.Stdout, "%s\n", path)
	return nil
}

var checkoutCmd = &cobra.Command{
	Use:     "checkout [-p problem-slug | -i problem-id] [-d difficulty] [-l language] [path]",
	Short:   "checkout a problem locally",
	Args:    cobra.MaximumNArgs(1),
	Example: `  tinycode checkout -d easy -l rust ./`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if difficultyStr != "" {
			if err := filters.AddFilter("difficulty", difficultyStr); err != nil {
				return err
			}
		}

		if statusStr != "" {
			if err := filters.AddFilter("status", statusStr); err != nil {
				return err
			}
		}

		if tagsStr != "" {
			if err := filters.AddFilter("tags", tagsStr); err != nil {
				return err
			}
		}

		if trackStr != "" {
			if err := filters.AddFilter("track", trackStr); err != nil {
				return err
			}
		}

		if _, err := filters.GetFilter("slug"); err != nil {
			log.Printf("no problem-slug provided, finding the next one")

			newFilters, err := client.FindNextChallenge(filters)

			if err != nil {
				return err
} else {
				filters.Update(&newFilters)
			}
		}

		if doSubmit {
			doOpen = true
		}

		if len(args) == 0 && doOpen {
			return fmt.Errorf("a source path must be provided when using --open")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		questionData, err := client.GetChallenge(filters)
		if err != nil {
			return err
		}

		var buf strings.Builder
		if err := provider.EncodeChallenge(backend, lang, filters, questionData, &buf); err != nil {
			return err
		}
		questionStr := buf.String()

		questionIdentity := questionData.Identify()

		if srcStr == "" {
			fmt.Fprintf(os.Stdout, "%s", questionStr)
		} else {
			stat, err := os.Stat(srcStr)
			if err == nil && stat.Mode().IsDir() {
				questionSlug, err := questionIdentity.GetFilter("slug")
				if err != nil {
					return err
				}

				ext := lang.Ext()

				var filename string
				switch lang.String() {
				case provider.Swift, provider.Java:
					filename = strcase.ToCamel(questionSlug)
				case provider.Rust:
					filename = strings.ReplaceAll(questionSlug, "-", "_")
				default:
					filename = questionSlug
				}

				srcStr = path.Join(srcStr, fmt.Sprintf("%s.%s", filename, ext))
			}

			toFileIfNotExists(srcStr, questionStr)

			srcDir := path.Dir(srcStr)
			files, err := questionData.Files()
			if err != nil {
				return err
			}

			var paths []string
			for name, content := range files {
				filepath := path.Join(srcDir, name)
				toFileIfNotExists(filepath, content)
				paths = append(paths, filepath)
			}

			if doOpen {
				editor := os.Getenv("EDITOR")
				if editor == "" {
					log.Fatal("no $EDITOR set, try `export EDITOR=emacs`")
				}

				editorCmd := exec.Command(editor, srcStr)
				if err := editorCmd.Start(); err != nil {
					return err
				}

				for _, path := range paths {
					open.Start(path)
				}

				if err := editorCmd.Wait(); err != nil {
					return err
				}

				if doSubmit {
					rootCmd.SetArgs([]string{"submit", srcStr})
					rootCmd.Execute()
				}
			}
		}

		return nil
	},
}
