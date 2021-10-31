package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func submissionFromReader(reader io.Reader, lang LangSlug) (*string, error) {
	buf := bytes.Buffer{}

	prefix, _, _, single, err := lang.Comment()
	if err != nil {
		return nil, err
	}

	if single == "" {
		single = prefix
	}

	regionBegin := fmt.Sprintf("%sleetcode submit region begin", single)
	regionEnd := fmt.Sprintf("%sleetcode submit region end", single)

	scanner := bufio.NewScanner(reader)

	const (
		SubmissionCode = iota
		Otherwise
	)

	mode := Otherwise
	for scanner.Scan() {
		line := string(scanner.Bytes())

		if strings.HasPrefix(line, regionBegin) {
			mode = SubmissionCode
			continue
		} else if strings.HasPrefix(line, regionEnd) {
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