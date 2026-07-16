package context

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Fingerprint returns a stable content digest after format-aware normalization.
func Fingerprint(kind string, data []byte) (string, error) {
	var normalized []byte
	var err error

	switch strings.ToLower(kind) {
	case "yaml", "yml":
		normalized, err = canonicalYAML(data)
	case "json":
		var value any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err = decoder.Decode(&value); err == nil {
			normalized, err = json.Marshal(value)
		}
	case "markdown", "md", "text", "txt":
		normalized = normalizeText(data)
	default:
		normalized = data
	}
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256(normalized)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func canonicalYAML(data []byte) ([]byte, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	if len(document.Content) != 1 {
		return nil, fmt.Errorf("yaml must contain exactly one document")
	}
	value, err := yamlValue(document.Content[0], "$")
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func yamlValue(node *yaml.Node, path string) (any, error) {
	switch node.Kind {
	case yaml.MappingNode:
		result := make(map[string]any, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			keyNode := node.Content[index]
			if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
				return nil, fmt.Errorf("mapping key at %s must be a string", path)
			}
			key := keyNode.Value
			if _, exists := result[key]; exists {
				return nil, fmt.Errorf("duplicate key %q at %s", key, path)
			}
			value, err := yamlValue(node.Content[index+1], path+"."+key)
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for index, child := range node.Content {
			value, err := yamlValue(child, fmt.Sprintf("%s[%d]", path, index))
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case yaml.ScalarNode:
		var value any
		if err := node.Decode(&value); err != nil {
			return nil, fmt.Errorf("decode scalar at %s: %w", path, err)
		}
		if text, ok := value.(string); ok {
			return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), nil
		}
		return value, nil
	case yaml.AliasNode:
		return nil, fmt.Errorf("yaml aliases are not supported at %s", path)
	default:
		return nil, fmt.Errorf("unsupported yaml node at %s", path)
	}
}

func normalizeText(data []byte) []byte {
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
