package admin

import (
	"context"
	"testing"
)

func TestAdminContextHasPermission(t *testing.T) {
	// Superuser context should have all permissions.
	superUserContext := context.WithValue(context.Background(), adminContextIsSuperUser, true)
	if !ContextHasPermission(superUserContext, 1) {
		t.Fatal("Expected superuser context to have all permissions")
	}

	// Context with specific permissions should report correctly.
	permIDs := []int64{1, 2, 3}
	permContext := context.WithValue(context.Background(), adminContextPermissions, permIDs)
	if !ContextHasPermission(permContext, 2) {
		t.Fatal("Expected context to have permission ID 2")
	}
	if ContextHasPermission(permContext, 4) {
		t.Fatal("Expected context to not have permission ID 4")
	}

	// Context with no permissions should report false.
	noPermContext := context.WithValue(context.Background(), adminContextPermissions, []int64{})
	if ContextHasPermission(noPermContext, 1) {
		t.Fatal("Expected context with no permissions to not have permission ID 1")
	}

	// Context with invalid permissions type should report false.
	invalidPermContext := context.WithValue(context.Background(), adminContextPermissions, "invalid")
	if ContextHasPermission(invalidPermContext, 1) {
		t.Fatal("Expected context with invalid permissions type to not have permission ID 1")
	}

	// Context with nil permissions should report false.
	nilPermContext := context.WithValue(context.Background(), adminContextPermissions, nil)
	if ContextHasPermission(nilPermContext, 1) {
		t.Fatal("Expected context with nil permissions to not have permission ID 1")
	}
}
