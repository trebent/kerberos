package env

import (
	"fmt"
	"os"
)

var validateFilePath = func(path string) error {
	if len(path) == 0 {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

var validatePort = func(v int) error {
	if v < 1000 || v > 65535 {
		return fmt.Errorf("must be between 1000 and 65535: %d", v)
	}
	return nil
}

var validateGreaterThanZero = func(v int) error {
	if v < 1 {
		return fmt.Errorf("must be greater than 0: %d", v)
	}
	return nil
}

var validateGreaterThanOrEqualToZero = func(v int) error {
	if v < 0 {
		return fmt.Errorf("must be greater than or equal to 0: %d", v)
	}
	return nil
}
