package syncer

import (
	"fmt"
	"strings"
)

type combinedErrors []error

func (errs combinedErrors) Error() string {
	var b strings.Builder
	b.WriteString("Collected errors:\n")
	for i, e := range errs {
		fmt.Fprintf(&b, "\tError %d: %s\n", i, e.Error())
	}

	return b.String()
}
