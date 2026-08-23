package security

import testlib "github.com/trebent/kerberos/test/lib"

type RequestEditorFn = testlib.RequestEditorFn

var (
	checkErr                    = testlib.CheckErr
	verifyStatusCode            = testlib.VerifyStatusCode
	verifyHeader                = testlib.VerifyHeader
	verifyHeaderMissing         = testlib.VerifyHeaderMissing
	getHost                     = testlib.GetHost
	getPort                     = testlib.GetPort
	getAdminPort                = testlib.GetAdminPort
	extractSessionCookie        = testlib.ExtractSessionCookie
	makeRequestEditorFromCookie = testlib.MakeRequestEditorFromCookie
)
