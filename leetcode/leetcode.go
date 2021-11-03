package leetcode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func cookiesFromString(cookies string) ([]*http.Cookie, error) {
	reqStr := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", cookies)
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
	return req.Cookies(), err
}

func cookieJarFromReader(reader io.Reader, url *url.URL) (http.CookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	allCookies, err := cookiesFromString(string(buf))
	if err != nil {
		return nil, err
	}

	var cookies []*http.Cookie
	for _, cookie := range allCookies {
		if cookie.Name == "csrftoken" || cookie.Name == "LEETCODE_SESSION" {
			cookies = append(cookies, cookie)
		}
	}

	jar.SetCookies(url, cookies)

	return jar, nil
}

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

func unmarshalFromResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		resp.Body.Close()

		return fmt.Errorf("invalid status code received from Client: %s\n%s", resp.Status, body)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	return json.Unmarshal(body, v)
}

type Client struct {
	raw  http.Client
	base *url.URL
}

func NewClient(cookieBuf io.Reader, base *url.URL) (*Client, error) {
	cookieJar, err := cookieJarFromReader(cookieBuf, base)
	if err != nil {
		return nil, err
	}

	client := http.Client{Transport: nil, CheckRedirect: nil, Jar: cookieJar}

	return &Client{client, base}, nil
}

func (client *Client) Do(method string, rawURL string, input interface{}, output interface{}) error {
	parsedURL, err := url.Parse(rawURL)

	if err != nil {
		return err
	}

	root, _ := url.Parse("/")
	baseUrl := parsedURL.ResolveReference(root)

	marshalled, err := json.Marshal(input)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, rawURL, bytes.NewBuffer(marshalled))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", rawURL)

	var csrf_token string
	for _, cookie := range client.raw.Jar.Cookies(baseUrl) {
		if cookie.Name == "csrftoken" {
			csrf_token = cookie.Value
		}
	}

	req.Header.Set("X-CSRFToken", csrf_token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.raw.Do(req)

	if err != nil {
		return err
	}

	return unmarshalFromResponse(resp, output)
}

func (client *Client) DoQuery(operationName string, query string, variables interface{}, output interface{}) error {
	graphQlRel, _ := url.Parse("/graphql")
	graphQlBase := client.base.ResolveReference(graphQlRel).String()

	type Query struct {
		OperationName string      `json:"operationName"`
		Query         string      `json:"query"`
		Variables     interface{} `json:"variables"`
	}

	req := Query{
		operationName,
		query,
		variables,
	}
	err := client.Do("POST", graphQlBase, req, output)
	return err
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

	if err := client.DoQuery("globalData", query, nil, &output); err != nil {
		return false, err
	} else {
		return output.Data.UserStatus.IsSignedIn, nil
	}
}

func (client *Client) GetRandomQuestion(filters Filters, categorySlug string) (string, error) {
	query := `
query randomQuestion($categorySlug: String, $filters: QuestionListFilterInput) {
  randomQuestion(categorySlug: $categorySlug, filters: $filters) {
    titleSlug
  }
}`

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
	if err := client.DoQuery("randomQuestion", query, variables, &output); err != nil {
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

func (client *Client) Submit(questionId string, slug string, lang LangSlug, code io.Reader) (*SubmitResponse, error) {
	submissionSrc, err := submissionFromReader(code, lang)
	if err != nil {
		return nil, err
	}

	path, err := url.Parse(fmt.Sprintf("/problems/%s/submit/", slug))
	if err != nil {
		return nil, err
	}

	submitUrl := client.base.ResolveReference(path).String()
	log.Printf("submit url: %s", submitUrl)

	submitRequest := SubmitRequest{Lang: lang, QuestionId: questionId, TypedCode: *submissionSrc}

	submitResp := SubmitResponse{}

	err = client.Do("POST", submitUrl, &submitRequest, &submitResp)
	if err != nil {
		return nil, err
	}

	submissionId := submitResp.SubmissionId
	log.Printf("successfully submitted solution: submissionId = %d", submissionId)

	return &submitResp, nil
}

func (client *Client) WaitUntilCompleteOrTimeOut(submissionId int64, timeOut time.Duration) (*CheckResponse, error) {
	checkPath, err := url.Parse(fmt.Sprintf("/submissions/detail/%d/check/", submissionId))
	if err != nil {
		return nil, err
	}

	checkUrl := client.base.ResolveReference(checkPath).String()

	backoff := 25 * time.Millisecond

	start := time.Now()
	for {
		checkResp := CheckResponse{}
		err := client.Do("GET", checkUrl, nil, &checkResp)
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

	err := client.DoQuery("questionData", query, variables, &res)
	if err != nil {
		return nil, err
	}

	return &res.Data.Question, nil
}
