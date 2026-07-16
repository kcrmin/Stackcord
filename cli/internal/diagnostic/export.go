package diagnostic

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Input contains diagnostics only; source files and prompts are deliberately excluded.
type Input struct {
	Versions       map[string]string
	Root, Home     string
	Errors         []string
	State          map[string]string
	Receipts       []string
	ProviderOutput string
}

var secret = regexp.MustCompile(`(?i)(api[_-]?token|token|password|secret|private[_-]?key)(=|:\s*)[^\s;&\"]+`)

// Export writes one privacy-safe diagnostic JSON entry to a portable ZIP archive.
func Export(writer io.Writer, input Input) error {
	archive := zip.NewWriter(writer)
	entry, err := archive.Create("diagnostic.json")
	if err != nil {
		return err
	}
	state := map[string]string{}
	keys := make([]string, 0, len(input.State))
	for key := range input.State {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		state[key] = sanitize(input.State[key], input)
	}
	errors := make([]string, len(input.Errors))
	for index, value := range input.Errors {
		errors[index] = sanitize(value, input)
	}
	receipts := make([]string, len(input.Receipts))
	for index, value := range input.Receipts {
		receipts[index] = sanitize(value, input)
	}
	sort.Strings(receipts)
	payload := map[string]any{"schema_version": 1, "versions": input.Versions, "errors": errors, "state": state, "receipts": receipts, "provider_status": sanitizeProvider(input.ProviderOutput)}
	encoder := json.NewEncoder(entry)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return err
	}
	return archive.Close()
}

func sanitize(value string, input Input) string {
	if input.Root != "" {
		value = strings.ReplaceAll(value, input.Root, "<PROJECT_ROOT>")
	}
	if input.Home != "" {
		value = strings.ReplaceAll(value, input.Home, "<HOME>")
	}
	value = redactURL(value)
	return secret.ReplaceAllString(value, "$1=[REDACTED]")
}

func sanitizeProvider(value string) string {
	if value == "" {
		return "not-provided"
	}
	// Provider output content is never exported; only its presence is useful.
	return "redacted-provider-output"
}

func redactURL(value string) string {
	for _, field := range strings.Fields(value) {
		candidate := strings.Trim(field, "\"',")
		parsed, err := url.Parse(candidate)
		if err == nil && parsed.User != nil {
			parsed.User = nil
			value = strings.ReplaceAll(value, candidate, parsed.String())
		}
	}
	return value
}
