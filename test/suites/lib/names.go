package lib

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"
)

var a atomic.Int32

func init() {
	a.Store(rand.Int32())
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
