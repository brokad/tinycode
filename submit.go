package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func submissionFromReader(reader io.Reader) (*string, error) {
	buf := bytes.Buffer{}

	scanner := bufio.NewScanner(reader)

	const (
		SubmissionCode = iota
		Otherwise
	)

	mode := Otherwise
	for scanner.Scan() {
		line := string(scanner.Bytes())

		if strings.HasPrefix(line, "//leetcode submit region begin") {
			mode = SubmissionCode
			continue
		} else if strings.HasPrefix(line, "//leetcode submit region end") {
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