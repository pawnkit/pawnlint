package output

import (
	"fmt"
	"io"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func writeCompact(w io.Writer, diags []diagnostic.Diagnostic) error {
	for _, d := range diags {
		pos := d.Range.Start
		if _, err := fmt.Fprintf(w, "%s:%d:%d: %s[%s]: %s\n", d.Filename, pos.Line, pos.Col, d.Severity, d.RuleID, d.Message); err != nil {
			return err
		}
	}
	return nil
}
