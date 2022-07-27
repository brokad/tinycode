package hackerrank

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"tinycode/provider"
)

func encodeFilters(filters map[string][]string) string {
	var params = url.Values{}
	for name, filter := range filters {
		for _, f := range filter {
			if f == "" {
				continue
			}
			params.Add(fmt.Sprintf("filters[%s][]", name), f)
		}
	}
	return params.Encode()
}

type Client struct {
	transport  provider.TransportClient
	DoPurchase bool // optional
}

func NewClient(base *url.URL) *Client {
	transport := provider.NewTransportClient(*base)
	return &Client{transport, false}
}

func (client *Client) Configure(config provider.BackendConfig) error {
	cookies := map[string]string{
		"csrftoken":      config.Csrf,
		"_hrank_session": config.Session,
	}

	if err := client.transport.SetCookies(cookies); err != nil {
		return err
	}

	client.transport.CsrfToken = config.Csrf
	client.transport.CsrfTokenHeader = config.CsrfHeader

	return nil
}

func (client *Client) GetLogIn() (string, string, error) {
	var csrf string
	var session string

	prefetchData, err := client.transport.ResolveReference("/prefetch_data")
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("GET", prefetchData.String(), nil)
	if err != nil {
		return "", "", err
	}

	type PrefetchResponse struct {
		CsrfToken string `json:"_csrf_token"`
	}

	resp, err := client.transport.RawDo(req)

	var sessionCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "_hrank_session" {
			session = cookie.Value
			sessionCookie = cookie
		}
	}

	respBody := PrefetchResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", "", err
	}

	csrf = respBody.CsrfToken

	if sessionCookie == nil {
		return "", "", fmt.Errorf("server response header did not set a session cookie")
	}

	type LoginRequest struct {
		Fallback   bool   `json:"fallback"`
		Login      string `json:"login"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}

	type LoginResponse struct {
		Status      bool     `json:"status"`
		Messages    []string `json:"messages"`
		Errors      []string `json:"errors"`
		ContestSlug string   `json:"contest_slug"`
		CsrfToken   string   `json:"csrf_token"`
	}

	var login string
	fmt.Print("hackerrank login: ")
	if _, err = fmt.Scanln(&login); err != nil {
		panic(err)
	}

	fmt.Print("hackerrank password: ")
	password, err := terminal.ReadPassword(0)
	if err != nil {
		panic(err)
	} else {
		fmt.Println()
	}

	loginReqUrl, err := client.transport.ResolveReference("/rest/auth/login")
	if err != nil {
		return "", "", err
	}

	loginReqBody := LoginRequest{
		Fallback:   false,
		Login:      login,
		Password:   string(password),
		RememberMe: false,
	}

	reqBuf, _ := json.Marshal(loginReqBody)

	req, _ = http.NewRequest("POST", loginReqUrl.String(), bytes.NewReader(reqBuf))

	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	baseUrl, _ := client.transport.ResolveReference("/")
	req.Header.Set("Host", baseUrl.Host)
	req.Header.Set("Origin", baseUrl.String())

	feLogin, _ := client.transport.ResolveReference("/auth/login")
	req.Header.Set("Referer", feLogin.String())

	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}

	log.Printf("Referer: %s, Host: %s, Origin: %s", feLogin.String(), baseUrl.Host, baseUrl.String())

	resp, err = client.transport.RawDo(req)
	if err != nil {
		return "", "", err
	}

	loginRespBody := LoginResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&loginRespBody); err != nil {
		return "", "", fmt.Errorf("could not decode server response: %s", err)
	}

	if loginRespBody.Status {
		csrf = loginRespBody.CsrfToken
		log.Printf("hackerrank: %s", strings.Join(loginRespBody.Messages, ": "))
	} else {
		return "", "", fmt.Errorf("error: %s", strings.Join(loginRespBody.Errors, ": "))
	}

	return csrf, session, nil
}

func (client *Client) GetChallengeData(contest string, challenge string) (*ChallengeData, error) {
	log.Printf("contest=%s challenge=%s", contest, challenge)

	parsedPath, err := url.Parse(fmt.Sprintf("/rest/contests/%s/challenges/%s/", contest, challenge))
	if err != nil {
		return nil, err
	}

	type GetChallengeResponse struct {
		Status bool          `json:"status"`
		Model  ChallengeData `json:"model"`
	}

	output := GetChallengeResponse{}
	if err := client.transport.Do("GET", parsedPath.String(), nil, &output); err != nil {
		return nil, err
	}

	return &output.Model, nil
}

func (client *Client) Do(method string, path string, req interface{}, output interface{}) error {
	type SubmitResponse struct {
		Model   json.RawMessage `json:"model"`
		Message string          `json:"message"`
	}

	var rawResp json.RawMessage

	if err := client.transport.Do(method, path, req, &rawResp); err != nil {
		return err
	}

	resp := SubmitResponse{}
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		log.Printf("unknown server response:\n%s", rawResp)
		return fmt.Errorf("could not unmarshal server response")
	}

	if resp.Message != "" {
		return fmt.Errorf("server-side error: %s", resp.Message)
	} else {
		if err := json.Unmarshal(resp.Model, output); err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) DoMany(method string, path string, req interface{}, output interface{}) error {
	type SubmitResponseMany struct {
		Models json.RawMessage `json:"models"`
		Total  uint64          `json:"total"`
	}

	resp := SubmitResponseMany{}

	if err := client.transport.Do(method, path, req, &resp); err != nil {
		return err
	}

	if err := json.Unmarshal(resp.Models, output); err != nil {
		return err
	}

	return nil
}

func (client *Client) DoSubmit(contest string, slug string, lang string, code string) (*SubmissionState, error) {
	parsedPath, err := url.Parse(fmt.Sprintf("/rest/contests/%s/challenges/%s/submissions", contest, slug))
	if err != nil {
		return nil, err
	}
	log.Printf("submit path: %s", parsedPath.String())

	req := SubmitRequest{
		Code:        code,
		ContestSlug: contest,
		Language:    lang,
	}

	state := SubmissionState{}

	if err := client.Do("POST", parsedPath.String(), &req, &state); err != nil {
		return nil, err
	}

	timeOut := 5 * time.Second
	backoff := 25 * time.Millisecond

	submissionUrl := fmt.Sprintf("%s/%d", parsedPath.String(), state.Id)
	log.Printf("submission path: %s", submissionUrl)

	start := time.Now()
	for {
		state := SubmissionState{}
		err := client.Do("GET", submissionUrl, nil, &state)
		if err != nil {
			return nil, err
		}

		if state.IsDone() {
			state.client = client
			return &state, nil
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

func (client *Client) GetUnlockedTestcases(contest string, challengeId int64) ([]int64, error) {
	checkPath := fmt.Sprintf("/rest/contests/%s/testcases/%d/all/unlocked_testcases", contest, challengeId)
	var unlockedCases []int64
	if err := client.transport.Do("GET", checkPath, nil, &unlockedCases); err != nil {
		return []int64{}, err
	} else {
		return unlockedCases, nil
	}
}

func (client *Client) HasUnlockedTestcase(contest string, challengeId int64, target int64) (bool, error) {
	unlockedCases, err := client.GetUnlockedTestcases(contest, challengeId)
	if err != nil {
		return false, err
	}

	for _, testcaseId := range unlockedCases {
		if testcaseId == target {
			return true, nil
		}
	}

	return false, nil
}

func (client *Client) PurchaseTestcaseData(contest string, challengeId int64, submissionId int64, testcaseId int64) (int64, error) {
	purchasePath := fmt.Sprintf(
		"/rest/contests/%s/testcases/%d/%d/purchase?submission_id=%d",
		contest,
		challengeId,
		testcaseId,
		submissionId,
	)

	type PurchaseResponse struct {
		HackoAmount int64 `json:"hacko_amount"`
	}

	resp := PurchaseResponse{}
	if err := client.transport.Do("GET", purchasePath, nil, &resp); err != nil {
		return -1, err
	} else {
		return resp.HackoAmount, nil
	}
}

func (client *Client) GetTestcaseData(contest string, challengeId int64, submissionId int64, testcaseId int64) (*TestcaseData, error) {
	unlocked, err := client.HasUnlockedTestcase(contest, challengeId, testcaseId)
	if err != nil {
		return nil, err
	}

	output := TestcaseData{}

	if !unlocked {
		if client.DoPurchase {
			left, err := client.PurchaseTestcaseData(contest, challengeId, submissionId, testcaseId)
			if err != nil {
				return nil, err
			}
			log.Printf("successfully purchased testcase %d: %d hackos left", testcaseId, left)
		} else {
			log.Printf("testcase %d not unlocked and will not purchase (no --purchase flag given)", testcaseId)
			return &output, nil
		}
	}

	path := fmt.Sprintf("/rest/contests/%s/testcases/%d/%d/testcase_data", contest, challengeId, testcaseId)
	if err := client.transport.Do("GET", path, nil, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (client *Client) IsSignedIn() (bool, error) {
	notifications := "/rest/contests/masters/notifications/summary"

	type NotificationsResponse struct {
		Status bool `json:"status"`
	}

	resp := &NotificationsResponse{}

	if err := client.transport.Do("GET", notifications, nil, &resp); err != nil {
		return false, err
	}

	return resp.Status, nil
}

func LocalizeLanguage(lang provider.Lang) (string, error) {
	return lang.String(), nil
}

func (client *Client) GetChallenge(filters provider.Filters) (provider.Challenge, error) {
	slug, err := filters.GetFilter("slug")
	if err != nil {
		return nil, err
	}

	contest, err := filters.GetFilter("contest")
	if err != nil {
		return nil, err
	}

	return client.GetChallengeData(contest, slug)
}

func (client *Client) ListChallenges(contest string, track string, offset uint64, limit uint64, filters map[string][]string) ([]ChallengeData, error) {
	path := fmt.Sprintf("/rest/contests/%s", contest)
	if track != "" {
		path = fmt.Sprintf("%s/tracks/%s", path, track)
	}

	type Values struct {
		Filters map[string][]string `url:"filters"`
	}

	path = fmt.Sprintf("%s/challenges?%s", path, encodeFilters(filters))
	log.Printf("list path: %s", path)

	var output []ChallengeData
	if err := client.DoMany("GET", path, nil, &output); err != nil {
		return nil, err
	} else {
		return output, nil
	}
}

func (client *Client) FindNextChallenge(filters provider.Filters) (provider.Filters, error) {
	var params = map[string][]string{}

	if difficulty, err := filters.GetFilter("difficulty"); err == nil {
		params["difficulty"] = []string{difficulty}
	}

	if subdomains, err := filters.GetFilter("tags"); err == nil {
		params["subdomains"] = []string{}
		for _, subdomain := range strings.Split(subdomains, ",") {
			params["subdomains"] = append(params["subdomains"], subdomain)
		}
	}

	if skills, err := filters.GetFilter("skills"); err == nil {
		params["skills"] = []string{}
		for _, skill := range strings.Split(skills, ",") {
			params["skill"] = append(params["skill"], skill)
		}
	}

	var output provider.Filters

	contest, err := filters.GetFilter("contest")
	if err != nil {
		return output, err
	}

	var track string
	if trackFilter, err := filters.GetFilter("track"); err == nil {
		track = trackFilter
	} else {
		// Not specifying a track explicitly leads to what seems to be a very
		// tough search for HackerRank's backend. So this is disabled in order
		// for us to be good citizens.
		return output, fmt.Errorf(`a --track is required: one of 
  algorithms
  data-structures
  mathematics
  ai
  c
  cpp
  java
  python
  ruby
  sql
  databases
  shell
  fp
  regex`)
	}

	if status, err := filters.GetFilter("status"); err == nil {
		params["status"] = []string{status}
	} else {
		params["status"] = []string{"unsolved"}
	}

	if challenges, err := client.ListChallenges(contest, track, 0, 1, params); err != nil {
		return output, err
	} else {
		if len(challenges) > 0 {
			return challenges[0].Identify(), nil
		} else {
			return output, fmt.Errorf("could not find challenge")
		}
	}
}

func (client *Client) Submit(filters provider.Filters, lang provider.Lang, code string) (provider.SubmissionReport, error) {
	slug, err := filters.GetFilter("slug")
	if err != nil {
		return nil, err
	}

	local, err := LocalizeLanguage(lang)
	if err != nil {
		return nil, err
	}

	contest, err := filters.GetFilter("contest")
	if err != nil {
		return nil, err
	}

	challenge, err := client.GetChallengeData(contest, slug)
	if err != nil {
		return nil, err
	}

	state, err := client.DoSubmit(contest, slug, local, code)
	if err != nil {
		return nil, err
	}

	state.maxScore = challenge.MaxScore

	return state, nil
}
