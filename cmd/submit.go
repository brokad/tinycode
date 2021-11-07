package cmd

import (
	"fmt"
	"github.com/brokad/tinycode/provider"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"io"
	"log"
	"math"
	"os"
	"strings"
)

func printSubmitReportAndExit(report provider.SubmissionReport) {
	if report.HasSucceeded() {
		log.Printf("%s: run succeeded", report.Identify())

		var stats = report.Statistics()

		var buf strings.Builder

		header := color.New(color.Bold, color.FgGreen)
		header.Fprintf(&buf, "Finished")

		if tc := stats.TotalTestCases; tc != 0 {
			fmt.Fprintf(&buf, " %d testcase", tc)
			if tc > 1 {
				fmt.Fprintf(&buf, "s")
			}
		}

		fmt.Fprintf(&buf, " done")

		if rt := stats.Runtime; rt != "" {
			fmt.Fprintf(&buf, " in %s", rt)
		}

		if rtp := stats.RuntimePercentile; !math.IsNaN(rtp) {
			fmt.Fprintf(&buf, " (better than %f%%)", rtp)
		}

		if mem := stats.Memory; mem != "" {
			fmt.Fprintf(&buf, " and using %s", mem)
		}

		if memp := stats.MemoryPercentile; !math.IsNaN(memp) {
			fmt.Fprintf(&buf, " (better than %f%%)", memp)
		}

		if score := stats.Score; score != "" {
			fmt.Fprintf(&buf, " and earned %s", score)
		}

		if maxs := stats.MaxScore; maxs != "" {
			fmt.Fprintf(&buf, " (out of %s)", maxs)
		}

		fmt.Fprintf(os.Stderr, "\n    %s", buf.String())

		os.Exit(0)
	} else {
		log.Printf("%s: run failed", report.Identify())

		errorReport := *report.ErrorReport()

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

		var lang *provider.Lang
		if langStr == "" {
			return fmt.Errorf("a --lang must be provided (e.g. rust)")
		} else {
			if lang, err = provider.ParseLang(langStr); err != nil {
				return err
			}
		}

		submitReport, err := client.Submit(filters, *lang, *code)
		if err != nil {
			return err
		}

		printSubmitReportAndExit(submitReport)

		return nil
	},
}