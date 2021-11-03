package cmd

import (
	"bufio"
	"fmt"
	"github.com/brokad/tinycode/leetcode"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

func printCheckResponseAndExit(res *leetcode.CheckResponse, submissionId int64) {
	if res.State != leetcode.Success {
		log.Fatal("checkResponse invalid: State != Success")
	}

	if res.HasSucceeded() {
		log.Printf("%d: run succeeded", submissionId)
		header := color.New(color.Bold, color.FgGreen)
		header.Fprintf(os.Stderr, "\n    Finished ")
		fmt.Fprintf(
			os.Stderr,
			"%d done in %s (%f%%) and %s (%f%%)\n\n",
			res.TotalTestCases,
			res.StatusRuntime,
			res.RuntimePercentile,
			res.StatusMemory,
			res.MemoryPercentile,
		)
		os.Exit(0)
	} else {
		log.Printf("%d: run failed", submissionId)

		var errorClass string
		var errorMsg string
		var ctxHeader string
		var ctxMsg string
		switch res.StatusCode {
		case leetcode.RuntimeError:
			errorClass = "runtime error"
			errorMsg = res.RuntimeError
			ctxHeader = fmt.Sprintf("last test case: %s", strings.ReplaceAll(res.LastTestCase, "\n", ", "))
			ctxMsg = fmt.Sprintf("expected output: %s\n\nruntime error: %s\n", res.ExpectedOutput, res.FullRuntimeError)
		case leetcode.CompileError:
			errorClass = "compile error"
			errorMsg = res.CompileError
			ctxMsg = fmt.Sprintf("%s\n", res.FullCompileError)
		case leetcode.WrongAnswer:
			errorClass = "wrong answer"
			errorMsg = "solution provided an invalid answer"
			ctxHeader = fmt.Sprintf("on input: %s", res.InputFormatted)
			ctxMsg = fmt.Sprintf("expected: %s\ngot: %s\n", res.ExpectedOutput, res.CodeOutput)
		case leetcode.TimeLimitExceeded:
			errorClass = "time limit exceeded"
			errorMsg = "solution took too long"
			ctxHeader = fmt.Sprintf("solution took: %dms", res.ElapsedTime)
			ctxMsg = fmt.Sprintf("on input: %s\nexpected output: %s\n", strings.ReplaceAll(res.LastTestCase, "\n", ", "), res.ExpectedOutput)
		default:
			errorClass = "unhandled"
			errorMsg = fmt.Sprintf("%s (%d)", res.StatusMsg, res.StatusCode)
			ctxMsg = fmt.Sprintf("%v", res)
		}

		header := color.New(color.Bold, color.FgRed)
		bold := color.New(color.Bold)
		ctx := color.New(color.FgCyan, color.Bold)

		var buf strings.Builder

		buf.WriteString(header.Sprintf(errorClass))
		buf.WriteString(bold.Sprintf(": %s\n", errorMsg))
		if ctxHeader != "" {
			buf.WriteString(ctx.Sprintf("  ---> "))
			buf.WriteString(fmt.Sprintln(ctxHeader))
		}
		if ctxMsg != "" {
			buf.WriteString(ctx.Sprintf("  | \n"))
			for _, line := range strings.Split(ctxMsg, "\n") {
				buf.WriteString(ctx.Sprintf("  | "))
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}

		fmt.Fprintf(os.Stderr, "\n%s\n", buf.String())
		os.Exit(int(res.StatusCode))
	}
}

var submitCmd = &cobra.Command{
	Use: "submit [-p problem-slug | -i problem-id] path",
	Short: "submit a solution to be judged",
	Args: cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if srcStr != "" {
			srcFile, err := os.Open(srcStr)
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(srcFile)
			if scanner.Scan() {
				fst := scanner.Text()
				re := regexp.MustCompile("leetcode metadata: question-id=([\\d]{1,4}) slug=([\\w-]+)")
				matches := re.FindStringSubmatch(fst)
				if len(matches) == 3 {
					problemId = matches[1]
					slug = matches[2]
				}
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if problemId == "" {
			log.Printf("a problem-id was not provided: attempting to query the api")
			questionData, err := client.GetQuestionData(slug)
			if err != nil {
				return err
			}

			problemId = questionData.QuestionId

			if problemId == "" {
				log.Fatal("a question-id must be provided")
			} else {
				log.Printf("found question-id=%s", problemId)
			}
		}

		var srcFile io.Reader

		if srcStr == "" {
			srcFile = os.Stdin
		} else {
			f, err := os.Open(srcStr)
			if err != nil {
				return err
			} else {
				srcFile = f
			}
		}

		submitResp, err := client.Submit(problemId, slug, lang, srcFile)
		if err != nil {
			return err
		}

		submissionId := submitResp.SubmissionId

		res, err := client.WaitUntilCompleteOrTimeOut(submissionId, 120*time.Second)
		if err != nil {
			return err
		}

		printCheckResponseAndExit(res, submissionId)

		return nil
	},
}