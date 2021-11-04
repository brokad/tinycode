package leetcode

import (
	"fmt"
	"github.com/brokad/tinycode/provider"
	"regexp"
	"strings"
)

type SubmitRequest struct {
	Lang       string `json:"lang"`
	QuestionId string `json:"question_id"`
	TypedCode  string `json:"typed_code"`
}

type SubmitResponse struct {
	SubmissionId int64 `json:"submission_id"`
}

type State string

const (
	Success State = "SUCCESS"
	Pending       = "PENDING"
)

type Status int16

const (
	Accepted            Status = 10
	WrongAnswer                = 11
	MemoryLimitExceeded        = 12
	OutputLimitExceeded        = 13
	TimeLimitExceeded          = 14
	RuntimeError               = 15
	InternalError              = 16
	CompileError               = 20
	UnknownError               = 21
	Unknown
)

type CheckResponse struct {
	StatusCode        Status  `json:"status_code"`
	Lang              string  `json:"lang"`
	RunSuccess        bool    `json:"run_success"`
	StatusRuntime     string  `json:"status_runtime"`
	CompileError      string  `json:"compile_error"`
	FullCompileError  string  `json:"full_compile_error"`
	RuntimeError      string  `json:"runtime_error"`
	FullRuntimeError  string  `json:"full_runtime_error"`
	Input             string  `json:"input"`
	InputFormatted    string  `json:"input_formatted"`
	Memory            int64   `json:"memory"`
	QuestionId        string  `json:"question_id"`
	ElapsedTime       uint64  `json:"elapsed_time"`
	CompareResult     string  `json:"compare_result"`
	CodeOutput        string  `json:"code_output"`
	StdOutput         string  `json:"std_output"`
	LastTestCase      string  `json:"last_testcase"`
	ExpectedOutput    string  `json:"expected_output"`
	TaskFinishTime    uint64  `json:"task_finish_time"`
	TotalCorrect      uint64  `json:"total_correct"`
	TotalTestCases    uint64  `json:"total_testcases"`
	RuntimePercentile float64 `json:"runtime_percentile"`
	StatusMemory      string  `json:"status_memory"`
	MemoryPercentile  float64 `json:"memory_percentile"`
	PrettyLang        string  `json:"pretty_lang"`
	SubmissionId      string  `json:"submission_id"`
	StatusMsg         string  `json:"status_msg"`
	State             State   `json:"state"`
}

func (res *CheckResponse) Summary() (*provider.SubmissionSummary, error) {
	var stats provider.SubmissionStatistics
	var errorReport *provider.ErrorReport
	if res.HasSucceeded() {
		stats.TotalTestCases = res.TotalTestCases
		stats.Runtime = res.StatusRuntime
		stats.RuntimePercentile = res.RuntimePercentile
		stats.Memory = res.StatusMemory
		stats.MemoryPercentile = res.MemoryPercentile
	} else {
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
		errorReport = &provider.ErrorReport{
			ErrorClass: errorClass,
			ErrorMsg:   errorMsg,
			CtxHeader:  ctxHeader,
			CtxMsg:     ctxMsg,
		}
	}
	return &provider.SubmissionSummary{Stats: stats, Error: errorReport}, nil
}

func (res *CheckResponse) HasSucceeded() bool {
	return res.StatusCode == Accepted && res.RunSuccess && res.TotalCorrect == res.TotalTestCases
}

func (res *CheckResponse) Identify() string {
	return res.SubmissionId
}

type CodeSnippet struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
}

type QuestionData struct {
	QuestionId   string        `json:"questionId"`
	Title        string        `json:"title"`
	TitleSlug    string        `json:"titleSlug"`
	Difficulty   string        `json:"difficulty"`
	Likes        uint64        `json:"likes"`
	Dislikes     uint64        `json:"dislikes"`
	Content      string        `json:"content"`
	CodeSnippets []CodeSnippet `json:"codeSnippets"`
}

type DifficultyFilter string

func ParseDifficulty(s string) (*DifficultyFilter, error) {
	var difficulty DifficultyFilter
	switch s {
	case "easy":
		difficulty = Easy
	case "medium":
		difficulty = Medium
	case "hard":
		difficulty = Hard
	case "":
		break
	default:
		return nil, fmt.Errorf("unknown difficulty: %s, must be one of: easy, medium, hard", s)
	}
	return &difficulty, nil
}

type StatusFilter string

func ParseStatus(s string) (*StatusFilter, error) {
	var status StatusFilter
	switch s {
	case "todo":
		status = Todo
	case "attempted":
		status = Attempted
	case "solved":
		status = Solved
	case "":
		break
	default:
		return nil, fmt.Errorf("unknown status: %s, must be one of: todo, attempted, solved", s)
	}
	return &status, nil
}

func (data *QuestionData) Identify() provider.Filters {
	var output provider.Filters
	if err := output.AddFilter("slug", data.TitleSlug); err != nil {
		panic(err)
	}

	if err := output.AddFilter("id", data.QuestionId); err != nil {
		panic(err)
	}

	return output
}

const (
	Easy   DifficultyFilter = "EASY"
	Medium                  = "MEDIUM"
	Hard                    = "HARD"
)

const (
	Todo      StatusFilter = "NOT_SOLVED"
	Solved                 = "AC"
	Attempted              = "TRIED"
)

type Filters struct {
	Difficulty DifficultyFilter `json:"difficulty,omitempty"`
	Status     StatusFilter     `json:"status,omitempty"`
	Tags       []string         `json:"tags,omitempty"`
}

func stripHtml(s string) string {
	re := regexp.MustCompile("<\\/?[^>]*>")
	output := re.ReplaceAllString(s, "")

	replacer := strings.NewReplacer(
		"&nbsp;", " ",
		"&lt;", "<",
		"&gt;", ">",
	)
	output = replacer.Replace(output)

	return output
}

func (data *QuestionData) Snippet(lang string) (string, error) {
	for _, snippet := range data.CodeSnippets {
		if snippet.LangSlug == lang {
			return snippet.Code, nil
		}
	}
	return "", fmt.Errorf("no snippet for lang %s found in server response, please try again", lang)
}

func (data *QuestionData) Prompt() string {
	return stripHtml(data.Content)
}
