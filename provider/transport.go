package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type TransportClient struct {
	raw             http.Client
	base            url.URL
	csrfToken       string
	csrfTokenHeader string
}

func NewTransportClient(jar http.CookieJar, base url.URL, csrfToken string, csrfTokenHeader string) TransportClient {
	raw := http.Client{Transport: nil, CheckRedirect: nil, Jar: jar}
	return TransportClient{
		raw,
		base,
		csrfToken,
		csrfTokenHeader,
	}
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

func (client *TransportClient) Do(method string, path string, input interface{}, output interface{}) error {
	parsedPath, err := url.Parse(path)
	if err != nil {
		return err
	}

	reqUrl := client.base.ResolveReference(parsedPath)

	marshalled, err := json.Marshal(input)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, reqUrl.String(), bytes.NewBuffer(marshalled))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", reqUrl.String())
	req.Header.Set("Host", client.base.Host)
	req.Header.Set(client.csrfTokenHeader, client.csrfToken)
	//req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.raw.Do(req)
	if err != nil {
		return err
	}

	return unmarshalFromResponse(resp, output)
}

func (client *TransportClient) DoQuery(operationName string, query string, variables interface{}, output interface{}) error {
	graphQlRel, _ := url.Parse("/graphql")
	graphQlBase := client.base.ResolveReference(graphQlRel)

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

	err := client.Do("POST", graphQlBase.String(), req, output)

	return err
}
