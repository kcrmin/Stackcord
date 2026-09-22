package output

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalogs/*.json
var catalogFiles embed.FS

// Message is localized human text with stable machine semantics.
type Message struct {
	Text       string `json:"text"`
	Severity   string `json:"severity"`
	DocSection string `json:"doc_section"`
}

// LoadCatalog returns an embedded catalog. Machine result codes remain English.
func LoadCatalog(locale string) (map[string]Message, error) {
	if locale != "ko" {
		locale = "en"
	}
	data, err := catalogFiles.ReadFile("catalogs/" + locale + ".json")
	if err != nil {
		return nil, fmt.Errorf("load %s catalog: %w", locale, err)
	}
	var catalog map[string]Message
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("decode %s catalog: %w", locale, err)
	}
	return catalog, nil
}

// Localize renders a message using explicit locale with English fallback.
func Localize(locale, key string, values map[string]string) string {
	catalog, _ := LoadCatalog(locale)
	message, exists := catalog[key]
	if !exists {
		english, _ := LoadCatalog("en")
		message = english[key]
	}
	text := message.Text
	for name, value := range values {
		text = strings.ReplaceAll(text, "{"+name+"}", value)
	}
	return text
}
