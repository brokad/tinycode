package provider

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Filters struct {
	raw map[string]string
}

func (filters *Filters) GetFilter(name string) (string, error) {
	if filters.raw == nil {
		filters.raw = map[string]string{}
	}
	if value, ok := filters.raw[name]; ok {
		return value, nil
	} else {
		return "", fmt.Errorf("a required filter was not provided: %s", name)
	}
}

func (filters *Filters) Update(other *Filters) {
	for k, v := range other.raw {
		_ = filters.AddFilter(k, v)
	}
}

func (filters *Filters) GetFilterOrDefault(name string) string {
	value, _ := filters.GetFilter(name)
	return value
}

func (filters *Filters) AddFilter(name string, value string) error {
	if validateFilterElement(name) && validateFilterElement(value) {
		if filters.raw == nil {
			filters.raw = map[string]string{}
		}
		filters.raw[name] = value
		return nil
	} else {
		return fmt.Errorf("not a valid filter value: %s: %s", name, value)
	}
}

func validateFilterElement(s string) bool {
	re := regexp.MustCompile("[\\w-]+")
	if matches := re.FindStringSubmatch(s); len(matches) == 1 {
		return matches[0] == s
	} else {
		return false
	}
}

func (filters *Filters) Render(buf io.StringWriter) error {
	for k, v := range filters.raw {
		if _, err := buf.WriteString(fmt.Sprintf("%s=%s ", k, v)); err != nil {
			return err
		}
	}
	return nil
}

func ParseFilters(s string) *Filters {
	var output Filters
	re := regexp.MustCompile("([\\w-]+)=([\\w-]+)")
	for _, matches := range re.FindAllStringSubmatch(s, -1) {		// Does not error handling because validation is ensured by the regex
		_ = output.AddFilter(matches[1], matches[2])
	}
	return &output
}

type Provider interface {
	IsSignedIn() (bool, error)
	LocalizeLanguage(Lang) (string, error)
	GetChallenge(Filters) (Challenge, error)
	FindNextChallenge(Filters) (Filters, error)
	Submit(Filters, string) (SubmissionReport, error)
}

type Challenge interface {
	Snippet(string) (string, error)
	Prompt() string
	Identify() Filters
}

type SubmissionReport interface {
	HasSucceeded() bool
	Identify() string
	Summary() (*SubmissionSummary, error)
}

type SubmissionSummary struct {
	Stats SubmissionStatistics
	Error *ErrorReport
}

type SubmissionStatistics struct {
	TotalTestCases    uint64
	Runtime           string
	RuntimePercentile float64
	Memory            string
	MemoryPercentile  float64
	Score             string
}

type ErrorReport struct {
	ErrorClass string
	ErrorMsg   string
	CtxHeader  string
	CtxMsg     string
}

type Lang struct {
	raw string
}

const (
	Cpp        string = "cpp"
	Java              = "java"
	Python            = "python"
	Python3           = "python3"
	C                 = "c"
	Csharp            = "csharp"
	JavaScript        = "javascript"
	Ruby              = "ruby"
	Swift             = "swift"
	Golang            = "golang"
	Scala             = "scala"
	Kotlin            = "kotlin"
	Rust              = "rust"
	Php               = "php"
	TypeScript        = "typescript"
	Racket            = "racket"
	Erlang            = "erlang"
	Elixir            = "elixir"
)

func ParseLang(s string) (*Lang, error) {
	switch s {
	case C,
		Cpp,
		Python,
		Python3,
		Csharp,
		JavaScript,
		Ruby,
		Swift,
		Golang,
		Scala,
		Kotlin,
		Rust,
		Php,
		TypeScript,
		Racket,
		Erlang,
		Elixir:
		return &Lang{raw: s}, nil
	default:
		return nil, fmt.Errorf("unknown or unsupported lang: %s", s)
	}
}

func (lang *Lang) String() string {
	return lang.raw
}

func (lang *Lang) Is(s string) bool {
	return lang.String() == s
}

func (lang *Lang) Comment() (string, string, string, string) {
	switch lang.raw {
	case C, Cpp, Java, Csharp, JavaScript, Swift, Golang, Scala, Kotlin, Php, TypeScript:
		return "/*", "*/", " * ", "// "
	case Python, Python3:
		return "\"\"\"", "\"\"\"", "   ", "# "
	case Ruby:
		return "=begin", "=end", "", "# "
	case Rust:
		return "//! ", "", "", "// "
	case Racket:
		return "#|", "|#", " ", "; "
	case Erlang:
		return "%", "", "", ""
	case Elixir:
		return "#", "", "", ""
	default:
		panic(fmt.Sprintf("unknown lang: %s", lang.raw))
	}
}

func (lang *Lang) Pretty() string {
	switch lang.raw {
	case Cpp:
		return "C++"
	case Java:
		return "Java"
	case Python:
		return "Python"
	case Python3:
		return "Python3"
	case C:
		return "C"
	case Csharp:
		return "C#"
	case JavaScript:
		return "JavaScript"
	case Ruby:
		return "Ruby"
	case Swift:
		return "Swift"
	case Golang:
		return "Go"
	case Scala:
		return "Scala"
	case Kotlin:
		return "Kotlin"
	case Rust:
		return "Rust"
	case Php:
		return "PHP"
	case TypeScript:
		return "TypeScript"
	case Racket:
		return "Racket"
	case Erlang:
		return "Erlang"
	case Elixir:
		return "Elixir"
	default:
		panic(fmt.Sprintf("unknown lang: %s", lang.raw))
	}
}

func ParseExt(ext string) (*Lang, error) {
	var raw string
	switch ext {
	case "cpp":
		raw = Cpp
	case "rs":
		raw = Rust
	case "swift":
		raw = Swift
	case "c":
		raw = C
	case "py":
		raw = Python3
	case "cs":
		raw = Csharp
	case "js":
		raw = JavaScript
	case "ts":
		raw = TypeScript
	case "rb":
		raw = Ruby
	case "go":
		raw = Golang
	case "scala", "sc":
		raw = Scala
	case "kt", "kts", "ktm":
		raw = Kotlin
	case "php":
		raw = Php
	case "erl":
		raw = Erlang
	case "ex", "exs":
		raw = Elixir
	case "rkt":
		raw = Racket
	default:
		return nil, fmt.Errorf("unrecognized file extension: %s", ext)
	}
	return &Lang{raw}, nil
}

func (lang *Lang) Ext() string {
	switch lang.raw {
	case Cpp:
		return "cpp"
	case Rust:
		return "rs"
	case Swift:
		return "swift"
	case Golang:
		return "go"
	case C:
		return "c"
	default:
		panic(fmt.Sprintf("unknown lang: %s", lang.raw))
	}
}

func EncodeChallenge(backend string, lang Lang, filters Filters, challenge Challenge, writer io.StringWriter) error {
	var headerBuf strings.Builder
	headerBuf.WriteString(backend)
	headerBuf.WriteString(" metadata: ")
	filters.Render(&headerBuf)
	headerBuf.WriteString("\n\n")
	headerBuf.WriteString(challenge.Prompt())
	header := headerBuf.String()

	prefix, suffix, perline, single := lang.Comment()

	if suffix == "" {
		perline = prefix
	}

	if prefix != "" {
		prefix += "\n"
	}

	if suffix != "" {
		suffix += "\n"
	}

	if suffix != "" {
		writer.WriteString(prefix)
	}

	// Add the header (metadata + formatted question statement)
	for _, line := range strings.Split(header, "\n") {
		writer.WriteString(fmt.Sprintf("%s%s\n", perline, line))
	}

	writer.WriteString(suffix)

	if lang.raw == Rust {
		// Switch to content comments from now on
		single = "// "
	}

	langStr, err := filters.GetFilter("lang")
	if err != nil {
		return err
	}

	// Add the solution prompt, braced by submission area brackets
	writer.WriteString(fmt.Sprintf("\n\n%s%s submit region begin\n", single, backend))
	if snippet, err := challenge.Snippet(langStr); err != nil {
		return err
	} else {
		writer.WriteString(snippet)
	}
	writer.WriteString(fmt.Sprintf("\n%s%s submit region end\n\n", single, backend))

	return nil
}

func DecodeSolution(backend string, reader io.Reader) (*string, error) {
	buf := bytes.Buffer{}

	regionBegin := fmt.Sprintf("%s submit region begin", backend)
	reBegin := regexp.MustCompile(regionBegin)

	regionEnd := fmt.Sprintf("%s submit region end", backend)
	reEnd := regexp.MustCompile(regionEnd)

	scanner := bufio.NewScanner(reader)

	const (
		SubmissionCode = iota
		Otherwise
	)

	mode := Otherwise
	for scanner.Scan() {
		line := string(scanner.Bytes())

		if reBegin.MatchString(line) {
			mode = SubmissionCode
			continue
		} else if reEnd.MatchString(line) {
			break
		}

		if mode == SubmissionCode {
			buf.WriteString(fmt.Sprintln(line))
		}
	}

	if mode != SubmissionCode {
		return nil, fmt.Errorf("provided source does not have a submission region")
	}

	output := buf.String()
	return &output, nil
}
