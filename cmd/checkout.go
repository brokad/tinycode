package cmd

import (
	"errors"
	"fmt"
	"github.com/brokad/tinycode/leetcode"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

var difficultyStr string
var statusStr string
var tagsStr string

var difficulty leetcode.DifficultyFilter
var status leetcode.StatusFilter
var tags []string

var doOpen bool

var checkoutCmd = &cobra.Command{
	Use:     "checkout [-p problem-slug | -i problem-id] [-d difficulty] [-l language] [path]",
	Short:   "checkout a problem locally",
	Args:    cobra.MaximumNArgs(1),
	Example: `  tinycode checkout -d easy -l rust ./`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		switch difficultyStr {
		case "easy":
			difficulty = leetcode.Easy
		case "medium":
			difficulty = leetcode.Medium
		case "hard":
			difficulty = leetcode.Hard
		case "":
			break
		default:
			return fmt.Errorf("unknown difficulty: %s, must be one of: easy, medium, hard", difficultyStr)
		}

		switch statusStr {
		case "todo":
			status = leetcode.Todo
		case "attempted":
			status = leetcode.Attempted
		case "solved":
			status = leetcode.Solved
		case "":
			break
		default:
			return fmt.Errorf("unknown status: %s, must be one of: todo, attempted, solved", statusStr)
		}

		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				tags = append(tags, tag)
			}
		}

		if slug == "" {
			log.Printf("no problem-slug provided, picking one at random")

			filters := leetcode.Filters{
				difficulty,
				status,
				tags,
			}

			chosenSlug, err := client.GetRandomQuestion(filters, "")
			if err != nil {
				return err
			} else {
				slug = chosenSlug
			}

			log.Printf("chose problem: %s", chosenSlug)
		}

		if len(args) == 0 && doOpen {
			return fmt.Errorf("a source path must be provided when using --open")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		questionData, err := client.GetQuestionData(slug)
		if err != nil {
			return err
		}

		questionStr, err := questionData.String(lang)
		if err != nil {
			return err
		}

		var output io.Writer
		if srcStr == "" {
			output = os.Stdout
		} else {
			stat, err := os.Stat(srcStr)

			if err == nil && stat.Mode().IsDir() {
				ext, err := lang.Ext()
				if err != nil {
					return err
				}
				srcStr = path.Join(srcStr, fmt.Sprintf("%s.%s", slug, ext))
				stat, err = os.Stat(srcStr)
			}

			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			} else if err == nil {
				log.Printf("file already exists: %s, not writing anything", srcStr)
				srcStr = os.DevNull
			}

			output, err = os.Create(srcStr)
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(output, "%s", *questionStr)

		if doOpen && srcStr != "" {
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

		return nil
	},
}
