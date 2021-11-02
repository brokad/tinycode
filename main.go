package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
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

func main() {
	log.SetPrefix("tinycode: ")
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	usr, err := user.Current()
	check(err)

	home := usr.HomeDir
	configPathDefault := path.Join(home, ".config/tinycode")

	backendStr := flag.String("backend", "leetcode", "which problem set provider to use (only leetcode allowed)")
	slugStr := flag.String("problem-slug", "", "the slug of the problem (e.g. two-sum), if not specified will pick a random problem")
	srcStr := flag.String("src", "", "the path to a source file (if not specified, uses stdin/stdout)")
	questionIdStr := flag.String("question-id", "", "the question id of the problem (e.g. 1)")
	configStr := flag.String("config", configPathDefault, "the path to the configuration directory")
	langStr := flag.String("lang", "", "the language of the submission (e.g. rust)")
	doSubmit := flag.Bool("submit", false, "whether to submit a solution (if not specified, will pull problem statement)")
	doOpen := flag.Bool("open", false, "whether to open the source file")

	doEasy := flag.Bool("easy", false, "")
	doMedium := flag.Bool("medium", false, "")
	doHard := flag.Bool("hard", false, "")

	doTodo := flag.Bool("todo", false, "")
	doAttempted := flag.Bool("attempted", false, "")
	doSolved := flag.Bool("solved", false, "")

	tagsStr := flag.String("tags", "", "")

	flag.Parse()

	var difficulty DifficultyFilter
	if *doEasy || *doMedium || *doHard {
		if *doEasy {
			difficulty = Easy
		} else if *doMedium {
			difficulty = Medium
		} else if *doHard {
			difficulty = Hard
		}
	}

	if difficulty != "" && *doSubmit {
		log.Printf("-easy/-medium/-hard ignored in -submit mode")
	}

	var status StatusFilter
	if *doTodo || *doAttempted || *doSolved {
		if *doTodo {
			status = Todo
		} else if *doAttempted {
			status = Attempted
		} else if *doSolved {
			status = Solved
		}
	}

	if status != "" && *doSubmit {
		log.Printf("-todo/-attempted/-solved ignored in -submit mode")
	}

	var tags []string
	if *tagsStr != "" {
		for _, tag := range strings.Split(*tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			tags = append(tags, tag)
		}
	}

	if len(tags) != 0 && *doSubmit {
		log.Printf("-tags ignored in -submit mode")
	}

	baseStr := new(string)
	switch *backendStr {
	case "leetcode":
		*baseStr = "https://leetcode.com"
	default:
		log.Fatalf("unknown provider: %s", *backendStr)
	}

	if *doOpen && *srcStr == "" {
		log.Fatal("-src must be set when using -open")
	}

	pwd, _ := os.Getwd()
	log.Printf("working directory: %s", pwd)

	base, err := url.Parse(*baseStr)
	check(err)

	cookieJarStr := new(string)
	*cookieJarStr = path.Join(*configStr, fmt.Sprintf("%s.cookies", *backendStr))
	cookieFile, err := os.Open(*cookieJarStr)
	if errors.Is(err, os.ErrNotExist) {
		printCookieReadmeAndExit(*backendStr, *cookieJarStr)
	}
	check(err)

	client, err := NewClient(cookieFile, base)
	check(err)

	if isSignedIn, err := client.IsSignedIn(); err != nil || !isSignedIn {
		if err != nil {
			log.Printf("error trying to check if user is signed in: %s", err)
		}
		if !isSignedIn {
			log.Printf("user is not signed in")
		}
		printCookieReadmeAndExit(*backendStr, *cookieJarStr)
	} else {
		log.Printf("valid authentication token found")
	}

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

		if !*doSubmit {
			log.Printf("no problem-slug provided, picking one at random")

			filters := Filters{
				difficulty,
				status,
				tags,
			}
			*slugStr, err = client.GetRandomQuestion(filters, "")
			check(err)

			log.Printf("chose problem: %s", *slugStr)
		}

		if *slugStr == "" {
			log.Fatal("a problem-slug must be provided")
		}
	}

	if *langStr == "" {
		if *srcStr != "" {
			spl := strings.Split(*srcStr, ".")
			if len(spl) > 1 {
				ext := spl[len(spl)-1]
				langSlug, err := NewLangFromExt(ext)
				*langStr = string(langSlug)
				check(err)
			}
		}

		if *langStr == "" {
			log.Fatal("a lang must be provided")
		}
	}

	langSlug := LangSlug(*langStr)

	if !*doSubmit {
		questionData, err := client.GetQuestionData(*slugStr)
		check(err)

		questionStr, err := questionData.String(langSlug)
		check(err)

		var output io.Writer
		if *srcStr == "" {
			output = os.Stdout
		} else {
			stat, err := os.Stat(*srcStr)

			if err == nil && stat.Mode().IsDir() {
				ext, err := langSlug.Ext()
				check(err)
				*srcStr = path.Join(*srcStr, fmt.Sprintf("%s.%s", *slugStr, ext))
				stat, err = os.Stat(*srcStr)
			}

			if err != nil && !errors.Is(err, os.ErrNotExist) {
				check(err)
			} else if err == nil {
				log.Printf("file already exists: %s, not writing anything", *srcStr)
				*srcStr = os.DevNull
			}

			output, err = os.Create(*srcStr)
			check(err)
		}

		fmt.Fprintf(output, "%s", *questionStr)

		if *doOpen && *srcStr != "" {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				log.Fatal("no $EDITOR set, try `export EDITOR=emacs`")
			}
			editorCmd := exec.Command(editor, *srcStr)
			check(editorCmd.Start())
			check(editorCmd.Wait())
		}
	} else { // doSubmit
		if *questionIdStr == "" {
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

		var srcFile io.Reader

		if *srcStr == "" {
			srcFile = os.Stdin
		} else {
			srcFile, err = os.Open(*srcStr)
			check(err)
		}

		submitResp, err := client.Submit(*questionIdStr, *slugStr, langSlug, srcFile)
		check(err)

		submissionId := submitResp.SubmissionId

		res, err := client.WaitUntilCompleteOrTimeOut(submissionId, 120*time.Second)
		check(err)

		printCheckResponseAndExit(res, submissionId)
	}
}
