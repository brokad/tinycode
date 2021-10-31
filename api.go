package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func (data *QuestionData) String(langSlug string) (*string, error) {
	header := fmt.Sprintf(
		"leetcode metadata: question-id=%s slug=%s\n\n%s",
		data.QuestionId,
		data.TitleSlug,
		stripHtml(data.Content),
	)

	var commentPrefix string
	switch langSlug {
	case "cpp":
		commentPrefix = "//"
	case "rust":
		commentPrefix = "//! "
	default:
		return nil, fmt.Errorf("langSlug unknown: %s", langSlug)
	}

	var buf strings.Builder

	// Add the header (metadata + formatted question statement)
	for _, line := range strings.Split(header, "\n") {
		buf.WriteString(fmt.Sprintf("%s%s\n", commentPrefix, line))
	}

	if langSlug == "rust" {
		// Switch to content comments from now on
		commentPrefix = "//"
	}

	// Add the solution prompt, braced by submission area brackets
	buf.WriteString(fmt.Sprintf("\n\n%sleetcode submit region begin(Prohibit modification and deletion)\n", commentPrefix))
	for _, snippet := range data.CodeSnippets {
		if snippet.LangSlug == langSlug {
			buf.WriteString(snippet.Code)
			buf.WriteString(fmt.Sprintf("\n%sleetcode submit region end(Prohibit modification and deletion)\n\n", commentPrefix))
			output := buf.String()
			return &output, nil
		}
	}

	return nil, fmt.Errorf("could not find snippet for langSlug=%s in leetcode response", langSlug)
}

func (r *CheckResponse) HasSucceeded() bool {
	return r.StatusCode == Accepted && r.RunSuccess && r.TotalCorrect == r.TotalTestCases
}

func unmarshalFromResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		resp.Body.Close()

		return fmt.Errorf("invalid status code received from leetcode: %s\n%s", resp.Status, body)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	return json.Unmarshal(body, v)
}
