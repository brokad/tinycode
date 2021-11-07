package hackerrank

import (
	"fmt"
	"github.com/brokad/tinycode/provider"
	"log"
	"reflect"
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
	BodyHtml    string   `json:"body_html"`
	Languages   []string `json:"languages"`
	Track       Track    `json:"track"`
	MaxScore    int64    `json:"max_score"`

	CTemplate     string `json:"c_template"`
	CTemplateHead string `json:"c_template_head"`
	CTemplateTail string `json:"c_template_tail"`

	CppTemplate     string `json:"cpp_template"`
	CppTemplateHead string `json:"cpp_template_head"`
	CppTemplateTail string `json:"cpp_template_tail"`

	JavaTemplate     string `json:"java_template"`
	JavaTemplateHead string `json:"java_template_head"`
	JavaTemplateTail string `json:"java_template_tail"`

	CsharpTemplate     string `json:"csharp_template"`
	CsharpTemplateHead string `json:"csharp_template_head"`
	CsharpTemplateTail string `json:"csharp_template_tail"`

	PhpTemplate     string `json:"php_template"`
	PhpTemplateHead string `json:"php_template_head"`
	PhpTemplateTail string `json:"php_template_tail"`

	RubyTemplate     string `json:"ruby_template"`
	RubyTemplateHead string `json:"ruby_template_head"`
	RubyTemplateTail string `json:"ruby_template_tail"`

	PythonTemplate     string `json:"python_template"`
	PythonTemplateHead string `json:"python_template_head"`
	PythonTemplateTail string `json:"python_template_tail"`

	PerlTemplate     string `json:"perl_template"`
	PerlTemplateHead string `json:"perl_template_head"`
	PerlTemplateTail string `json:"perl_template_tail"`

	HaskellTemplate     string `json:"haskell_template"`
	HaskellTemplateHead string `json:"haskell_template_head"`
	HaskellTemplateTail string `json:"haskell_template_tail"`

	ClojureTemplate     string `json:"clojure_template"`
	ClojureTemplateHead string `json:"clojure_template_head"`
	ClojureTemplateTail string `json:"clojure_template_tail"`

	ScalaTemplate     string `json:"scala_template"`
	ScalaTemplateHead string `json:"scala_template_head"`
	ScalaTemplateTail string `json:"scala_template_tail"`

	LuaTemplate     string `json:"lua_template"`
	LuaTemplateHead string `json:"lua_template_head"`
	LuaTemplateTail string `json:"lua_template_tail"`

	ErlangTemplate     string `json:"erlang_template"`
	ErlangTemplateHead string `json:"erlang_template_head"`
	ErlangTemplateTail string `json:"erlang_template_tail"`

	JavascriptTemplate     string `json:"javascript_template"`
	JavascriptTemplateHead string `json:"javascript_template_head"`
	JavascriptTemplateTail string `json:"javascript_template_tail"`

	TypescriptTemplate     string `json:"typescript_template"`
	TypescriptTemplateHead string `json:"typescript_template_head"`
	TypescriptTemplateTail string `json:"typescript_template_tail"`

	GoTemplate     string `json:"go_template"`
	GoTemplateHead string `json:"go_template_head"`
	GoTemplateTail string `json:"go_template_tail"`

	Python3Template     string `json:"python3_template"`
	Python3TemplateHead string `json:"python3_template_head"`
	Python3TemplateTail string `json:"python3_template_tail"`

	ObjectivecTemplate     string `json:"objectivec_template"`
	ObjectivecTemplateHead string `json:"objectivec_template_head"`
	ObjectivecTemplateTail string `json:"objectivec_template_tail"`

	Java8Template     string `json:"java8_template"`
	Java8TemplateHead string `json:"java8_template_head"`
	Java8TemplateTail string `json:"java8_template_tail"`

	SwiftTemplate     string `json:"swift_template"`
	SwiftTemplateHead string `json:"swift_template_head"`
	SwiftTemplateTail string `json:"swift_template_tail"`

	Cpp14Template     string `json:"cpp14_template"`
	Cpp14TemplateHead string `json:"cpp14_template_head"`
	Cpp14TemplateTail string `json:"cpp14_template_tail"`

	PypyTemplate     string `json:"pypy_template"`
	PypyTemplateHead string `json:"pypy_template_head"`
	PypyTemplateTail string `json:"pypy_template_tail"`

	Pypy3Template     string `json:"pypy3_template"`
	Pypy3TemplateHead string `json:"pypy3_template_head"`
	Pypy3TemplateTail string `json:"pypy3_template_tail"`

	KotlinTemplate     string `json:"kotlin_template"`
	KotlinTemplateHead string `json:"kotlin_template_head"`
	KotlinTemplateTail string `json:"kotlin_template_tail"`

	Java15Template     string `json:"java15_template"`
	Java15TemplateHead string `json:"java15_template_head"`
	Java15TemplateTail string `json:"java15_template_tail"`
}

func (data *ChallengeData) promptHtmlFilename() string {
	return fmt.Sprintf("%s.html", data.Slug)
}

func (data *ChallengeData) Snippet(lang provider.Lang) (string, error) {
	var head string
	var template string
	var tail string

	local, err := LocalizeLanguage(lang)
	if err != nil {
		return "", err
	}

	v := reflect.ValueOf(*data)
	rootName := fmt.Sprintf("%s_template", local)
	headName := fmt.Sprintf("%s_head", rootName)
	tailName := fmt.Sprintf("%s_tail", rootName)

	for i := 0; i < v.NumField(); i++ {
		name, ok := v.Type().Field(i).Tag.Lookup("json")
		if !ok {
			continue
		}

		value := v.Field(i)

		switch name {
		case rootName:
			template = value.String()
		case headName:
			head = value.String()
		case tailName:
			tail = value.String()
		}
	}

	output := fmt.Sprintf("%s%s%s", head, template, tail)

	if output != "" {
		return output, nil
	} else {
		return output, fmt.Errorf("no snippet for lang %s (hackerrank %s) found in server response", lang, local)
	}
}

func (data *ChallengeData) Prompt() string {
	return fmt.Sprintf("For instructions open: %s", data.promptHtmlFilename())
}

func (data *ChallengeData) Files() (map[string]string, error) {
	return map[string]string{
		data.promptHtmlFilename(): data.BodyHtml,
	}, nil
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

func (state *SubmissionState) ErrorReport() *provider.ErrorReport {
	if state.HasSucceeded() {
		return nil
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
		if err == nil {
			if testcaseData.Stdin == "" {
				testcaseData.Stdin = "[paywalled, use the --purchase flag to unlock]"
			}

			if testcaseData.ExpectedOutput == "" {
				testcaseData.ExpectedOutput = "[paywalled]"
			}

			output.CtxHeader = fmt.Sprintf("last test case: %s", testcaseData.Stdin)

			output.CtxMsg = fmt.Sprintf("expected output: %s\n", testcaseData.ExpectedOutput)
		} else {
			log.Printf("could not retrieve testcase data: %s", err)
		}
	}

	return &output
}

func (state *SubmissionState) Statistics() provider.SubmissionStatistics {
	var stats = provider.NewStatistics()

	if !state.IsDone() {
		log.Printf("submission is processing: %d, no statistics yet", state.Id)
		return stats
	}

	var totalRuntime = 0.

	for _, time := range state.CodecheckerTime {
		totalRuntime += time
	}

	if totalRuntime != 0 {
		stats.Runtime = fmt.Sprintf("%fms", totalRuntime*100)
	}

	stats.TotalTestCases = uint64(len(state.IndividualTestcaseScore))
	stats.Score = fmt.Sprintf("%spts", state.DisplayScore)

	if state.maxScore > 0 {
		stats.MaxScore = fmt.Sprintf("%dpts", state.maxScore)
	}

	return stats
}
