package integration

import testlib "github.com/trebent/kerberos/test/lib"

type RequestEditorFn = testlib.RequestEditorFn

var (
	checkErr                    = testlib.CheckErr
	verifyStatusCode            = testlib.VerifyStatusCode
	verifyHeader                = testlib.VerifyHeader
	verifyHeaderMissing         = testlib.VerifyHeaderMissing
	extractSessionCookie        = testlib.ExtractSessionCookie
	extractRefreshCookie        = testlib.ExtractRefreshCookie
	makeRequestEditorFromCookie = testlib.MakeRequestEditorFromCookie
	getAdminPort                = testlib.GetAdminPort
	getPort                     = testlib.GetPort
	getHost                     = testlib.GetHost
	getMetricsPort              = testlib.GetMetricsPort
	getJaegerAPIPort            = testlib.GetJaegerAPIPort
)
