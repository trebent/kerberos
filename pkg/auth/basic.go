package auth

import (
	"context"
	"net/http"

	"github.com/trebent/kerberos/internal/auth/method/basic"
)

type (
	UserKey   struct{}
	OrgKey    struct{}
	GroupsKey struct{}
)

// ContextMiddleware extracts request information and populates the request context.
func ContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = setUser(ctx, r)
			ctx = setOrg(ctx, r)
			ctx = setGroups(ctx, r)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns the user ID from the given context if one exists. If a value is found,
// value, true is returned, otherwise an empty string and false.
func UserFromContext(ctx context.Context) (string, bool) {
	return extractFromContext(ctx, UserKey{})
}

// OrgFromContext returns the org ID from the given context if one exists. If a value is found,
// value, true is returned, otherwise an empty string and false.
func OrgFromContext(ctx context.Context) (string, bool) {
	return extractFromContext(ctx, OrgKey{})
}

// GroupsFromContext returns the groups from the given context if one exists. If a value is found,
// value, true is returned, otherwise an empty string and false.
func GroupsFromContext(ctx context.Context) (string, bool) {
	return extractFromContext(ctx, GroupsKey{})
}

func extractFromContext(ctx context.Context, key any) (string, bool) {
	val := ctx.Value(key)
	if val == nil {
		return "", false
	}

	//nolint:errcheck // tightly controlled
	return val.(string), true
}

func setUser(ctx context.Context, r *http.Request) context.Context {
	if val := r.Header.Get(basic.HeaderUser); val != "" {
		ctx = context.WithValue(ctx, UserKey{}, val)
	}

	return ctx
}

func setOrg(ctx context.Context, r *http.Request) context.Context {
	if val := r.Header.Get(basic.HeaderUser); val != "" {
		ctx = context.WithValue(ctx, OrgKey{}, val)
	}

	return ctx
}

func setGroups(ctx context.Context, r *http.Request) context.Context {
	if val := r.Header.Get(basic.HeaderUser); val != "" {
		ctx = context.WithValue(ctx, GroupsKey{}, val)
	}

	return ctx
}
