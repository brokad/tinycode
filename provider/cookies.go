package provider

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func cookiesFromString(cookies string) ([]*http.Cookie, error) {
	reqStr := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", cookies)
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
	return req.Cookies(), err
}

func CookieJarFromReader(reader io.Reader, url *url.URL, filter []string) (http.CookieJar, error) {
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
		for _, name := range filter {
			if cookie.Name == name {
				cookies = append(cookies, cookie)
			}
		}
	}

	jar.SetCookies(url, cookies)

	return jar, nil
}