package lib

import (
	"fmt"
	"sync/atomic"
)

var a atomic.Int32

// InitNames seeds the atomic counter used to generate unique names.
// Call this from TestMain with a random seed.
func InitNames(seed int32) {
	a.Store(seed)
}

// Username returns a guaranteed unique username.
func Username() string {
	return fmt.Sprintf("Smith-%d", a.Add(1))
}

// OrgName returns a guaranteed unique organisation name.
func OrgName() string {
	return fmt.Sprintf("Org-%d", a.Add(1))
}

// GroupName returns a guaranteed unique group name.
func GroupName() string {
	return fmt.Sprintf("Group-%d", a.Add(1))
}
