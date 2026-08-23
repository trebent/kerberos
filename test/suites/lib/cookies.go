package lib

import (
	"context"
	"fmt"
	"net/http"
)

type RequestEditorFn func(ctx context.Context, req *http.Request) error

func ExtractSessionCookie(resp *http.Response) (*http.Cookie, error) {
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in response")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		return nil, fmt.Errorf("session cookie not found in response")
	}

	return sessionCookie, nil
}

func ExtractRefreshCookie(resp *http.Response) (*http.Cookie, error) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "refresh" {
			return cookie, nil
		}
	}

	return nil, fmt.Errorf("refresh cookie not found in response")
}

func MakeRequestEditorFromCookie(cookie *http.Cookie) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.AddCookie(cookie)
		return nil
	}
}
