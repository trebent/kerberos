package lib

import (
	"fmt"
	"net/http"
	"time"

	adminapi "github.com/trebent/kerberos/pkg/admin/client"
	authbasicapi "github.com/trebent/kerberos/pkg/auth/client"
)

const (
	SuperUserClientID     = "admin"
	SuperUserClientSecret = "secret"
)

var (
	Client = &http.Client{Timeout: 4 * time.Second}

	AdminClient, _ = adminapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d", GetHost(), GetAdminPort()),
	)
	BasicAuthClient, _ = authbasicapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d", GetHost(), GetAdminPort()),
	)
)
