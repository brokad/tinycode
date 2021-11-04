package cmd

import (
	"errors"
	"fmt"
	"github.com/brokad/tinycode/provider"
	"github.com/iancoleman/strcase"
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

var doOpen bool

var checkoutCmd = &cobra.Command{
	Use:     "checkout [-p problem-slug | -i problem-id] [-d difficulty] [-l language] [path]",
	Short:   "checkout a problem locally",
	Args:    cobra.MaximumNArgs(1),
	Example: `  tinycode checkout -d easy -l rust ./`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if difficultyStr != "" {
			filters.AddFilter("difficulty", difficultyStr)
		}

		if statusStr != "" {
			filters.AddFilter("status", statusStr)
		}

		if tagsStr != "" {
			filters.AddFilter("tags", tagsStr)
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

			stat, err = os.Stat(srcStr)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			} else if err == nil {
				log.Printf("file already exists: %s, not writing anything", srcStr)
			} else {
				log.Printf("writing to file %s", srcStr)

				output, err := os.Create(srcStr)
				if err != nil {
					return err
				}
				fmt.Fprintf(output, "%s", questionStr)
				fmt.Fprintf(os.Stdout, "%s", srcStr)
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

				if err := editorCmd.Wait(); err != nil {
					return err
				}
			}
		}

		return nil
	},
}
