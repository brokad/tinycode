package leetcode

import (
	"fmt"
	"regexp"
	"strings"
)

type SubmitRequest struct {
	Lang       LangSlug `json:"lang"`
	QuestionId string   `json:"question_id"`
	TypedCode  string   `json:"typed_code"`
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

type LangSlug string

const (
	Cpp        LangSlug = "cpp"
	Java                = "java"
	Python              = "python"
	Python3             = "python3"
	C                   = "c"
	Csharp              = "csharp"
	JavaScript          = "javascript"
	Ruby                = "ruby"
	Swift               = "swift"
	Golang              = "golang"
	Scala               = "scala"
	Kotlin              = "kotlin"
	Rust                = "rust"
	Php                 = "php"
	TypeScript          = "typescript"
	Racket              = "racket"
	Erlang              = "erlang"
	Elixir              = "elixir"
)

func (slug *LangSlug) Pretty() (string, error) {
	switch *slug {
	case Cpp:
		return "C++", nil
	case Java:
		return "Java", nil
	case Python:
		return "Python", nil
	case Python3:
		return "Python3", nil
	case C:
		return "C", nil
	case Csharp:
		return "C#", nil
	case JavaScript:
		return "JavaScript", nil
	case Ruby:
		return "Ruby", nil
	case Swift:
		return "Swift", nil
	case Golang:
		return "Go", nil
	case Scala:
		return "Scala", nil
	case Kotlin:
		return "Kotlin", nil
	case Rust:
		return "Rust", nil
	case Php:
		return "PHP", nil
	case TypeScript:
		return "TypeScript", nil
	case Racket:
		return "Racket", nil
	case Erlang:
		return "Erlang", nil
	case Elixir:
		return "Elixir", nil
	default:
		return "", fmt.Errorf("unsupported lang slug: %s", slug)
	}
}

func (slug *LangSlug) Comment() (string, string, string, string, error) {
	switch *slug {
	case C, Cpp, Java, Csharp, JavaScript, Swift, Golang, Scala, Kotlin, Php, TypeScript:
		return "/*", "*/", " * ", "// ", nil
	case Python, Python3:
		return "\"\"\"", "\"\"\"", "   ", "# ", nil
	case Ruby:
		return "=begin", "=end", "", "# ", nil
	case Rust:
		return "//! ", "", "", "// ", nil
	case Racket:
		return "#|", "|#", " ", "; ", nil
	case Erlang:
		return "%", "", "", "", nil
	case Elixir:
		return "#", "", "", "", nil
	default:
		return "", "", "", "", fmt.Errorf("unsupported lang slug: %s", slug)
	}
}

func NewLangFromExt(ext string) (LangSlug, error) {
	switch ext {
	case "cpp":
		return Cpp, nil
	case "rs":
		return Rust, nil
	case "swift":
		return Swift, nil
	case "c":
		return C, nil
	case "py":
		return Python3, nil
	case "cs":
		return Csharp, nil
	case "js":
		return JavaScript, nil
	case "ts":
		return TypeScript, nil
	case "rb":
		return Ruby, nil
	case "go":
		return Golang, nil
	case "scala", "sc":
		return Scala, nil
	case "kt", "kts", "ktm":
		return Kotlin, nil
	case "php":
		return Php, nil
	case "erl":
		return Erlang, nil
	case "ex", "exs":
		return Elixir, nil
	case "rkt":
		return Racket, nil
	default:
		return "", fmt.Errorf("unrecognized file extension: %s", ext)
	}
}

func (slug *LangSlug) Ext() (string, error) {
	switch *slug {
	case Cpp:
		return "cpp", nil
	case Rust:
		return "rs", nil
	case Swift:
		return "swift", nil
	case Golang:
		return "go", nil
	default:
		return "", fmt.Errorf("unrecognized lang slug: %s", *slug)
	}
}

type CheckResponse struct {
	StatusCode        Status   `json:"status_code"`
	Lang              LangSlug `json:"lang"`
	RunSuccess        bool     `json:"run_success"`
	StatusRuntime     string   `json:"status_runtime"`
	CompileError      string   `json:"compile_error"`
	FullCompileError  string   `json:"full_compile_error"`
	RuntimeError      string   `json:"runtime_error"`
	FullRuntimeError  string   `json:"full_runtime_error"`
	Input             string   `json:"input"`
	InputFormatted    string   `json:"input_formatted"`
	Memory            int64    `json:"memory"`
	QuestionId        string   `json:"question_id"`
	ElapsedTime       uint64   `json:"elapsed_time"`
	CompareResult     string   `json:"compare_result"`
	CodeOutput        string   `json:"code_output"`
	StdOutput         string   `json:"std_output"`
	LastTestCase      string   `json:"last_testcase"`
	ExpectedOutput    string   `json:"expected_output"`
	TaskFinishTime    uint64   `json:"task_finish_time"`
	TotalCorrect      uint64   `json:"total_correct"`
	TotalTestCases    uint64   `json:"total_testcases"`
	RuntimePercentile float64  `json:"runtime_percentile"`
	StatusMemory      string   `json:"status_memory"`
	MemoryPercentile  float64  `json:"memory_percentile"`
	PrettyLang        string   `json:"pretty_lang"`
	SubmissionId      string   `json:"submission_id"`
	StatusMsg         string   `json:"status_msg"`
	State             State    `json:"state"`
}

type CodeSnippet struct {
	Lang     string   `json:"lang"`
	LangSlug LangSlug `json:"langSlug"`
	Code     string   `json:"code"`
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

const (
	Easy   DifficultyFilter = "EASY"
	Medium                  = "MEDIUM"
	Hard                    = "HARD"
)

type StatusFilter string

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

func (data *QuestionData) String(langSlug LangSlug) (*string, error) {
	header := fmt.Sprintf(
		"Client metadata: question-id=%s slug=%s\n\n\n%s:\n\n%s",
		data.QuestionId,
		data.TitleSlug,
		data.Title,
		stripHtml(data.Content),
	)

	prefix, suffix, perline, single, err := langSlug.Comment()
	if err != nil {
		return nil, err
	}

	if suffix == "" {
		perline = prefix
	}

	if prefix != "" {
		prefix += "\n"
	}

	if suffix != "" {
		suffix += "\n"
	}

	var buf strings.Builder

	if suffix != "" {
		buf.WriteString(prefix)
	}

	// Add the header (metadata + formatted question statement)
	for _, line := range strings.Split(header, "\n") {
		buf.WriteString(fmt.Sprintf("%s%s\n", perline, line))
	}

	buf.WriteString(suffix)

	if langSlug == Rust {
		// Switch to content comments from now on
		single = "// "
	}

	// Add the solution prompt, braced by submission area brackets
	buf.WriteString(fmt.Sprintf("\n\n%sleetcode submit region begin\n", single))
	for _, snippet := range data.CodeSnippets {
		if snippet.LangSlug == langSlug {
			buf.WriteString(snippet.Code)
			buf.WriteString(fmt.Sprintf("\n%sleetcode submit region end\n\n", single))
			output := buf.String()
			return &output, nil
		}
	}

	return nil, fmt.Errorf("could not find snippet for langSlug=%s in Client response", langSlug)
}

func (r *CheckResponse) HasSucceeded() bool {
	return r.StatusCode == Accepted && r.RunSuccess && r.TotalCorrect == r.TotalTestCases
}
