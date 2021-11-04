package leetcode

import (
	"fmt"
	"github.com/brokad/tinycode/provider"
	"io"
	"log"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	transport provider.TransportClient
}

func NewClient(cookieBuf io.Reader, base *url.URL) (*Client, error) {
	cookieJar, err := provider.CookieJarFromReader(cookieBuf, base, []string{"csrftoken", "LEETCODE_SESSION"})
	if err != nil {
		return nil, err
	}

	var csrfToken string
	for _, cookie := range cookieJar.Cookies(base) {
		if cookie.Name == "csrftoken" {
			csrfToken = cookie.Value
		}
	}

	if csrfToken == "" {
		return nil, fmt.Errorf("could not find csrftoken cookie: try logging in to leetcode again")
	}

	transport := provider.NewTransportClient(cookieJar, *base, csrfToken)

	return &Client{transport}, nil
}

func (client *Client) IsSignedIn() (bool, error) {
	query := `
query globalData {
  userStatus {
    isSignedIn
  }
}`
	type UserStatus struct {
		IsSignedIn bool `json:"isSignedIn"`
	}

	type QueryData struct {
		UserStatus UserStatus `json:"userStatus"`
	}

	type QueryResult struct {
		Data QueryData `json:"data"`
	}

	output := QueryResult{}

	if err := client.transport.DoQuery("globalData", query, nil, &output); err != nil {
		return false, err
	} else {
		return output.Data.UserStatus.IsSignedIn, nil
	}
}

func (client *Client) GetRandomQuestionSlug(difficulty DifficultyFilter, status StatusFilter, tags []string, categorySlug string) (string, error) {
	query := `
query randomQuestion($categorySlug: String, $filters: QuestionListFilterInput) {
  randomQuestion(categorySlug: $categorySlug, filters: $filters) {
    titleSlug
  }
}`

	filters := Filters {
		difficulty,
		status,
		tags,
	}

	type Variables struct {
		CategorySlug string  `json:"categorySlug"`
		Filters      Filters `json:"filters"`
	}

	variables := Variables{
		categorySlug,
		filters,
	}

	type RandomQuestionData struct {
		TitleSlug string `json:"titleSlug"`
	}

	type QueryData struct {
		RandomQuestion RandomQuestionData `json:"randomQuestion"`
	}

	type QueryResult struct {
		Data QueryData `json:"data"`
	}

	output := QueryResult{}
	if err := client.transport.DoQuery("randomQuestion", query, variables, &output); err != nil {
		return "", err
	} else {
		titleSlug := output.Data.RandomQuestion.TitleSlug
		if titleSlug == "" {
			return "", fmt.Errorf("could not find a viable question, try removing conditions")
		} else {
			return titleSlug, nil
		}
	}
}

func (client *Client) FindNextChallenge(filters provider.Filters) (provider.Filters, error) {
	var err error
	var output provider.Filters

	difficultyStr := filters.GetFilterOrDefault("difficulty")
	difficulty, err := ParseDifficulty(difficultyStr)
	if err != nil {
		return output, err
	}

	statusStr := filters.GetFilterOrDefault("status")
	status, err := ParseStatus(statusStr)
	if err != nil {
		return output, err
	}

	var tags []string
	tagsStr := filters.GetFilterOrDefault("tags")

	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			tags = append(tags, tag)
		}
	}

	questionSlug, err := client.GetRandomQuestionSlug(*difficulty, *status, tags, "")
	if err != nil {
		return output, err
	}

	if err := output.AddFilter("slug", questionSlug); err != nil {
		return output, err
	} else {
		return output, nil
	}
}

func (client *Client) SubmitCode(questionId string, slug string, lang string, code string) (*SubmitResponse, error) {
	submitPath, err := url.Parse(fmt.Sprintf("/problems/%s/submit/", slug))
	if err != nil {
		return nil, err
	}

	log.Printf("submit path: %s", submitPath)

	submitRequest := SubmitRequest{Lang: lang, QuestionId: questionId, TypedCode: code}

	submitResp := SubmitResponse{}

	err = client.transport.Do("POST", submitPath.String(), &submitRequest, &submitResp)
	if err != nil {
		return nil, err
	}

	submissionId := submitResp.SubmissionId
	log.Printf("successfully submitted solution: submissionId = %d", submissionId)

	return &submitResp, nil
}

func (client *Client) Submit(filters provider.Filters, code string) (provider.SubmissionReport, error) {
	questionId, err := filters.GetFilter("id")
	if err != nil {
		return nil, err
	}

	slug, err := filters.GetFilter("slug")
	if err != nil {
		return nil, err
	}

	langStr, err := filters.GetFilter("lang")
	if err != nil {
		return nil, err
	}

	submitResponse, err := client.SubmitCode(questionId, slug, langStr, code)
	if err != nil {
		return nil, err
	}

	submissionId := submitResponse.SubmissionId

	return client.WaitUntilCompleteOrTimeOut(submissionId, 120*time.Second)
}

func (client *Client) WaitUntilCompleteOrTimeOut(submissionId int64, timeOut time.Duration) (*CheckResponse, error) {
	checkPath, err := url.Parse(fmt.Sprintf("/submissions/detail/%d/check/", submissionId))
	if err != nil {
		return nil, err
	}

	backoff := 25 * time.Millisecond

	start := time.Now()
	for {
		checkResp := CheckResponse{}
		err := client.transport.Do("GET", checkPath.String(), nil, &checkResp)
		if err != nil {
			return nil, err
		}

		if checkResp.State == Success {
			return &checkResp, nil
		}

		// Wait a bit before trying again
		backoff *= 2
		if time.Now().Add(backoff).Before(start.Add(timeOut)) {
			time.Sleep(backoff)
		} else {
			break
		}
	}

	return nil, fmt.Errorf("request timed out after %s", timeOut)
}

func (client *Client) GetQuestionData(titleSlug string) (*QuestionData, error) {
	query := `
query questionData($titleSlug: String!) {
  question(titleSlug: $titleSlug) {
    questionId
    title
    titleSlug
    content
    difficulty
    likes
    dislikes
    exampleTestcases
    codeSnippets {
      lang
      langSlug
      code
      __typename
    }
    sampleTestCase
    metaData
    envInfo
    __typename
  }
}
`
	variables := map[string]string{
		"titleSlug": titleSlug,
	}

	type ResponseResult struct {
		Question QuestionData `json:"question"`
	}

	type QueryResult struct {
		Data ResponseResult `json:"data"`
	}

	res := QueryResult{}

	err := client.transport.DoQuery("questionData", query, variables, &res)
	if err != nil {
		return nil, err
	}

	return &res.Data.Question, nil
}

func (client *Client) GetChallenge(filters provider.Filters) (provider.Challenge, error) {
	if slug, err := filters.GetFilter("slug"); err == nil {
		return client.GetQuestionData(slug)
	} else {
		return nil, err
	}
}

func (client *Client) LocalizeLanguage(lang provider.Lang) (string, error) {
	return lang.String(), nil
}