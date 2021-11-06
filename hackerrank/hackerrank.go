package hackerrank

import (
	"encoding/json"
	"fmt"
	"github.com/brokad/tinycode/provider"
	"log"
	"net/url"
	"strings"
	"time"
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

func NewClient(config provider.BackendConfig, base *url.URL) (*Client, error) {
	cookies := map[string]string{
		"csrftoken":      config.Csrf,
		"_hrank_session": config.Auth,
	}
	cookieJar, err := provider.CookieJarFromMap(cookies, base)
	if err != nil {
		return nil, err
	}

	transport := provider.NewTransportClient(cookieJar, *base, config.Csrf, config.CsrfHeader)

	return &Client{transport, false}, nil
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

func (client *Client) LocalizeLanguage(lang provider.Lang) (string, error) {
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
		// tough search for Hackerrank's backend. So this is disabled in order
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

func (client *Client) Submit(filters provider.Filters, code string) (provider.SubmissionReport, error) {
	slug, err := filters.GetFilter("slug")
	if err != nil {
		return nil, err
	}

	lang, err := filters.GetFilter("lang")
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

	state, err := client.DoSubmit(contest, slug, lang, code)
	if err != nil {
		return nil, err
	}

	state.maxScore = challenge.MaxScore

	return state, nil
}
