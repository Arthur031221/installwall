package registry

import "fmt"

func httpStatusErr(code int) error {
	return fmt.Errorf("unexpected status %d", code)
}
