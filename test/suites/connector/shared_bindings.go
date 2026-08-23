package connector

import testlib "github.com/trebent/kerberos/test/lib"

type RequestEditorFn = testlib.RequestEditorFn

var (
	checkErr                    = testlib.CheckErr
	verifyStatusCode            = testlib.VerifyStatusCode
	verifyHeader                = testlib.VerifyHeader
	verifyHeaderMissing         = testlib.VerifyHeaderMissing
	extractSessionCookie        = testlib.ExtractSessionCookie
	makeRequestEditorFromCookie = testlib.MakeRequestEditorFromCookie
	getHost                     = testlib.GetHost
	getAdminPort                = testlib.GetAdminPort
	getConnectorPort            = testlib.GetConnectorPort
)
