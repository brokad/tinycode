package hackerrank

import (
	"fmt"
	"github.com/brokad/tinycode/provider"
	"strings"
)

type Track struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	TrackId   int64  `json:"track_id"`
	TrackName string `json:"track_name"`
	TrackSlug string `json:"track_slug"`
}

type ChallengeData struct {
	Solved      bool     `json:"solved"`
	Attempted   bool     `json:"attempted"`
	ContestSlug string   `json:"contest_slug"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Preview     string   `json:"preview"`
	Category    string   `json:"category"`
	Languages   []string `json:"languages"`
	Track       Track    `json:"track"`
	CTemplate   string   `json:"c_template"`
	MaxScore    int64    `json:"max_score"`
}

func (data *ChallengeData) Snippet(lang string) (string, error) {
	var template string
	switch lang {
	case "c":
		template = data.CTemplate
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}

	if template == "" {
		return "", fmt.Errorf("no snippet for language in server response: %s", lang)
	}
	return template, nil
}

func (data *ChallengeData) Prompt() string {
	return ""
}

func (data *ChallengeData) Identify() provider.Filters {
	var output = provider.Filters{}
	output.AddFilter("slug", data.Slug)
	output.AddFilter("category", data.Category)
	output.AddFilter("contest", data.ContestSlug)
	return output
}

type SubmitRequest struct {
	Code         string `json:"code"`
	ContestSlug  string `json:"contest_slug"`
	Language     string `json:"language"`
	PlaylistSlug string `json:"playlist_slug"` // optional
}

type TestcaseData struct {
	Stdin          string `json:"stdin"`
	ExpectedOutput string `json:"expected_output"`
}

type Status string

const (
	Accepted         Status = "Accepted"
	Success                 = "Success"
	Processing              = "Processing"
	CompilationError        = "Compilation error"
	RuntimeError            = "Runtime Error"
	TimeoutError            = "Terminated due to timeout"
)

type SubmissionState struct {
	Id                      int64     `json:"id"`
	ContestId               int64     `json:"contest_id"`
	ChallengeId             int64     `json:"challenge_id"`
	Language                string    `json:"language"`
	Status                  Status    `json:"status"`
	ChallengeSlug           string    `json:"challenge_slug"`
	ContestSlug             string    `json:"contest_slug"`
	CompileStatus           int64     `json:"compile_status"`
	CompileMessage          string    `json:"compile_message"`
	TestcaseStatus          []int64   `json:"testcase_status"`
	TestcaseMessage         []string  `json:"testcase_message"`
	CodecheckerTime         []float64 `json:"codecheck_time"`
	CodecheckerSignal       []int64   `json:"codechecker_signal"`
	IndividualTestcaseScore []float64 `json:"individual_test_case_score"`
	DisplayScore            string    `json:"display_score"`
	maxScore                int64
	client                  *Client
}

func (state *SubmissionState) IsDone() bool {
	return state.Status != Processing
}

func (state *SubmissionState) HasSucceeded() bool {
	return state.Status == Accepted || state.Status == Success
}

func (state *SubmissionState) Identify() string {
	return fmt.Sprintf(
		"/rest/contests/%s/challenges/%s/submissions/%d",
		state.ContestSlug,
		state.ChallengeSlug,
		state.Id,
	)
}

func (state *SubmissionState) findFirstFailedTestcase() int {
	for idx, msg := range state.TestcaseMessage {
		if msg != Success {
			return idx
		}
	}
	return -1
}

func (state *SubmissionState) ErrorReport() (*provider.ErrorReport, error) {
	if state.HasSucceeded() {
		return nil, nil
	}

	output := provider.ErrorReport{
		ErrorClass: string(state.Status),
	}

	if state.CompileStatus != 0 {

		msgSplit := strings.Split(state.CompileMessage, ":")

		if len(msgSplit) > 0 {
			output.ErrorMsg = msgSplit[0]
		}

		if len(msgSplit) > 1 {
			output.CtxHeader = msgSplit[1]
		}

		if len(msgSplit) > 2 {
			output.CtxMsg = strings.Join(msgSplit[2:], "\n")
		}
	} else {
		// Find the first failed testcase
		firstFailedIdx := state.findFirstFailedTestcase()
		if firstFailedIdx == -1 {
			panic(fmt.Sprintf("submission has no failed testcase and yet has not succeeded: %v", *state))
		}

		output.ErrorMsg = fmt.Sprintf("Test Case %d: %s", firstFailedIdx, state.TestcaseMessage[firstFailedIdx])

		testcaseData, err := state.client.GetTestcaseData(state.ContestSlug, state.ChallengeId, state.Id, int64(firstFailedIdx))
		if err != nil {
			return nil, err
		}

		if testcaseData.Stdin == "" {
			testcaseData.Stdin = "[paywalled, use --purchase flag to unlock]"
		}

		if testcaseData.ExpectedOutput == "" {
			testcaseData.ExpectedOutput = "[paywalled]"
		}

		output.CtxHeader = fmt.Sprintf("last test case: %s", testcaseData.Stdin)
		output.CtxMsg = fmt.Sprintf("expected output: %s\n", testcaseData.ExpectedOutput)
	}

	return &output, nil
}

func (state *SubmissionState) Summary() (*provider.SubmissionSummary, error) {
	if !state.IsDone() {
		return nil, fmt.Errorf("submission is processing: %d", state.Id)
	}

	var totalRuntime = 0.

	for _, time := range state.CodecheckerTime {
		totalRuntime += time
	}

	runtime := fmt.Sprintf("%fms", totalRuntime*100)

	score := state.DisplayScore

	stats := provider.SubmissionStatistics{
		TotalTestCases: uint64(len(state.IndividualTestcaseScore)),
		Runtime:        runtime,
		Score:          score,
	}

	output := provider.SubmissionSummary{
		Stats: stats,
	}

	if !state.HasSucceeded() {
		if e, err := state.ErrorReport(); err != nil {
			return nil, err
		} else {
			output.Error = e
		}
	}

	return &output, nil
}
