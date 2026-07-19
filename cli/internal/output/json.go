package output

import (
	"encoding/json"
	"io"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

// WriteJSON encodes one normalized command result and a trailing newline.
func WriteJSON(w io.Writer, result domain.Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(result.Normalize())
}
