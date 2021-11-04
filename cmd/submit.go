package cmd

import (
	"bufio"
	"fmt"
	"github.com/brokad/tinycode/provider"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

func printSubmitReportAndExit(report provider.SubmissionReport) {
	summary, _ := report.Summary()
	if report.HasSucceeded() {
		log.Printf("%s: run succeeded", report.Identify())
		header := color.New(color.Bold, color.FgGreen)
		header.Fprintf(os.Stderr, "\n    Finished ")
		fmt.Fprintf(
			os.Stderr,
			"%d done in %s (better than %f%%) and %s (better than %f%%)\n\n",
			summary.Stats.TotalTestCases,
			summary.Stats.Runtime,
			summary.Stats.RuntimePercentile,
			summary.Stats.Memory,
			summary.Stats.MemoryPercentile,
		)
		os.Exit(0)
	} else {
		log.Printf("%s: run failed", report.Identify())

		errorReport := *summary.Error

		header := color.New(color.Bold, color.FgRed)
		bold := color.New(color.Bold)
		ctx := color.New(color.FgCyan, color.Bold)

		var buf strings.Builder
		buf.WriteString(header.Sprintf(errorReport.ErrorClass))
		buf.WriteString(bold.Sprintf(": %s\n", errorReport.ErrorMsg))
		if errorReport.CtxHeader != "" {
			buf.WriteString(ctx.Sprintf("  ---> "))
			buf.WriteString(fmt.Sprintln(errorReport.CtxHeader))
		}
		if errorReport.CtxMsg != "" {
			buf.WriteString(ctx.Sprintf("  | \n"))
			for _, line := range strings.Split(errorReport.CtxMsg, "\n") {
				buf.WriteString(ctx.Sprintf("  | "))
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}
		output := buf.String()

		fmt.Fprintf(os.Stderr, "\n%s\n", output)
		os.Exit(1)
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
			for scanner.Scan() {
				ln := scanner.Text()
				re := regexp.MustCompile("([\\w-]+) metadata: ")
				matches := re.FindStringSubmatch(ln)
				if len(matches) != 0 {
					backend = matches[1]
					log.Printf("metadata found for %s", backend)
					parsedFilters := provider.ParseFilters(ln)
					filters.Update(parsedFilters)
				}
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		challenge, err := client.GetChallenge(filters)
		if err != nil {
			return err
		}

		challengeFilters := challenge.Identify()
		filters.Update(&challengeFilters)

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

		code, err := provider.DecodeSolution(backend, srcFile)
		if err != nil {
			return err
		}

		submitReport, err := client.Submit(filters, *code)
		if err != nil {
			return err
		}

		printSubmitReportAndExit(submitReport)

		return nil
	},
}