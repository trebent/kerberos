package security

import (
	"net/http"
	"testing"

	testlib "github.com/trebent/kerberos/test/lib"
)

type RequestEditorFn = testlib.RequestEditorFn

func checkErr(err error, t *testing.T) {
	testlib.CheckErr(err, t)
}

func verifyStatusCode(in int, expected int, t *testing.T) {
	testlib.VerifyStatusCode(in, expected, t)
}

func verifyHeader(headers http.Header, key string, expectedValue string, t *testing.T) {
	testlib.VerifyHeader(headers, key, expectedValue, t)
}

func verifyHeaderMissing(headers http.Header, key string, t *testing.T) {
	testlib.VerifyHeaderMissing(headers, key, t)
}

func getHost() string {
	return testlib.GetHost()
}

func getPort() int {
	return testlib.GetPort()
}

func getAdminPort() int {
	return testlib.GetAdminPort()
}

func extractSessionCookie(resp *http.Response) (*http.Cookie, error) {
	return testlib.ExtractSessionCookie(resp)
}

func makeRequestEditorFromCookie(cookie *http.Cookie) RequestEditorFn {
	return testlib.MakeRequestEditorFromCookie(cookie)
}
