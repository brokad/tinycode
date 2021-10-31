package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func printCheckResponseAndExit(res *CheckResponse, submissionId int64) {
	if res.State != Success {
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
		case RuntimeError:
			errorClass = "runtime error"
			errorMsg = res.RuntimeError
			ctxHeader = fmt.Sprintf("last test case: %s", strings.ReplaceAll(res.LastTestCase, "\n", ", "))
			ctxMsg = fmt.Sprintf("expected output: %s\n\nruntime error: %s\n", res.ExpectedOutput, res.FullRuntimeError)
		case CompileError:
			errorClass = "compile error"
			errorMsg = res.CompileError
			ctxMsg = fmt.Sprintf("%s\n", res.FullCompileError)
		case WrongAnswer:
			errorClass = "wrong answer"
			errorMsg = "solution provided an invalid answer"
			ctxHeader = fmt.Sprintf("on input: %s", res.InputFormatted)
			ctxMsg = fmt.Sprintf("expected: %s\ngot: %s\n", res.ExpectedOutput, res.CodeOutput)
		case TimeLimitExceeded:
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

func main() {
	log.SetPrefix("leetcode: ")
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)

	baseStr := flag.String("base", "https://leetcode.com", "the leetcode base URL")
	slugStr := flag.String("slug", "", "the slug of the problem")
	srcStr := flag.String("src", "", "the path to the source file")
	questionIdStr := flag.String("question-id", "", "the question id of the problem")
	cookieJarStr := flag.String("cookie-jar", "./cookie-jar", "the path to the cookie jar file")
	langStr := flag.String("lang", "", "the language of the submission")
	doSubmit := flag.Bool("submit", false, "whether to submit a solution")

	flag.Parse()

	pwd, _ := os.Getwd()
	log.Printf("working directory: %s", pwd)

	base, err := url.Parse(*baseStr)
	check(err)

	if *slugStr == "" {
		if *srcStr != "" && *doSubmit {
			srcFile, err := os.Open(*srcStr)
			check(err)

			scanner := bufio.NewScanner(srcFile)
			if scanner.Scan() {
				fst := scanner.Text()
				re := regexp.MustCompile("leetcode metadata: question-id=([\\d]{1,4}) slug=([\\w-]+)")
				matches := re.FindStringSubmatch(fst)
				if len(matches) == 3 {
					*questionIdStr = matches[1]
					*slugStr = matches[2]
				}
			}
		}

		if *slugStr == "" {
			log.Fatal("a problem slug must be provided")
		}
	}

	cookieFile, err := os.Open(*cookieJarStr)
	check(err)

	client, err := NewClient(cookieFile, base)
	check(err)

	if *questionIdStr == "" && *doSubmit {
		log.Printf("a question-id was not provided: attempting to query the api")
		questionData, err := client.GetQuestionData(*slugStr)
		check(err)

		*questionIdStr = questionData.QuestionId

		if *questionIdStr == "" {
			log.Fatal("a question-id must be provided")
		} else {
			log.Printf("found question-id=%s", *questionIdStr)
		}
	}

	if *langStr == "" {
		if *srcStr != "" {
			spl := strings.Split(*srcStr, ".")
			if len(spl) > 1 {
				ext := spl[len(spl) - 1]
				switch ext {
				case "cpp":
					*langStr = "cpp"
				case "rs":
					*langStr = "rust"
				default:
					log.Fatalf("unrecognized file extension: %s, a lang must be provided", ext)
				}
			}
		}

		if *langStr == "" {
			log.Fatal("a lang must be provided")
		}
	}

	if ! *doSubmit {
		questionData, err := client.GetQuestionData(*slugStr)
		check(err)

		questionStr, err := questionData.String(*langStr)
		check(err)

		var output io.Writer
		if *srcStr == "" {
			output = os.Stdout
		} else {
			output, err = os.Create(*srcStr)
			check(err)
		}

		fmt.Fprintf(output, "%s", *questionStr)
	} else {
		var srcFile io.Reader

		if *srcStr == "" {
			srcFile = os.Stdin
		} else {
			srcFile, err = os.Open(*srcStr)
			check(err)
		}

		submitResp, err := client.Submit(*questionIdStr, *slugStr, *langStr, srcFile)
		check(err)

		submissionId := submitResp.SubmissionId

		res, err := client.WaitUntilCompleteOrTimeOut(submissionId, 120*time.Second)
		check(err)

		printCheckResponseAndExit(res, submissionId)
	}
}
