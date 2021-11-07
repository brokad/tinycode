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
	CsrfToken       string
	CsrfTokenHeader string
}

func NewTransportClient(base url.URL) TransportClient {
	raw := http.Client{Transport: nil, CheckRedirect: nil, Jar: nil}
	return TransportClient{
		raw: raw,
		base: base,
		CsrfToken: "",
		CsrfTokenHeader: "",
	}
}

func (client *TransportClient) ResolveReference(path string) (*url.URL, error) {
	if parsed, err := url.Parse(path); err != nil {
		return nil, err
	} else {
		return client.base.ResolveReference(parsed), nil
	}
}

func (client *TransportClient) SetCookieJar(jar http.CookieJar) {
	client.raw.Jar = jar
}

func (client *TransportClient) SetCookies(cookies map[string]string) error {
	jar, err := CookieJarFromMap(cookies, &client.base)
	if err != nil {
		return err
	} else {
		client.SetCookieJar(jar)
		return nil
	}
}

func unmarshalFromResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		resp.Body.Close()

		return fmt.Errorf("error from server: %s, body: %s", resp.Status, body)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	return json.Unmarshal(body, v)
}

func (client *TransportClient) RawDo(r *http.Request) (*http.Response, error) {
	return client.raw.Do(r)
}

func (client *TransportClient) Do(method string, path string, input interface{}, output interface{}) error {
	reqUrl, err := client.ResolveReference(path)
	if err != nil {
		return err
	}

	marshalled, err := json.Marshal(input)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, reqUrl.String(), bytes.NewReader(marshalled))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", reqUrl.String())
	req.Header.Set("Host", client.base.Host)
	req.Header.Set("Origin", client.base.String())

	log.Printf("Referer: %s, Host: %s, Origin: %s", reqUrl.String(), client.base.Host, client.base.String())

	if client.CsrfToken != "" {
		req.Header.Set(client.CsrfTokenHeader, client.CsrfToken)
	} else {
		return fmt.Errorf("client has not set a CSRF token")
	}

	resp, err := client.RawDo(req)
	if err != nil {
		return err
	}

	return unmarshalFromResponse(resp, output)
}

func (client *TransportClient) DoQuery(operationName string, query string, variables interface{}, output interface{}) error {
	type Query struct {
		OperationName string      `json:"operationName"`
		Query         string      `json:"query"`
		Variables     interface{} `json:"variables,omitempty"`
	}

	req := Query{
		operationName,
		query,
		variables,
	}

	err := client.Do("POST", "/graphql", req, output)

	return err
}
