package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

// DecodeJSON strictly decodes exactly one JSON document with duplicate and unknown fields rejected.
func DecodeJSON[T any](data []byte) (T, error) {
	var zero T
	duplicateCheck := json.NewDecoder(bytes.NewReader(data))
	duplicateCheck.UseNumber()
	if err := scanJSONValue(duplicateCheck, "$"); err != nil {
		return zero, err
	}
	if _, err := duplicateCheck.Token(); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("expected exactly one JSON document")
		}
		return zero, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("expected exactly one JSON document")
		}
		return zero, err
	}
	return value, nil
}

func scanJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key at %s must be a string", path)
			}
			if seen[key] {
				return fmt.Errorf("duplicate key %q at %s", key, path)
			}
			seen[key] = true
			if err := scanJSONValue(decoder, path+"."+key); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
			index++
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

// LoadYAML decodes exactly one YAML document with duplicate and unknown fields rejected.
func LoadYAML[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", path, err)
	}
	value, err := DecodeYAML[T](data)
	if err != nil {
		return zero, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

// DecodeYAML strictly decodes one in-memory YAML document.
func DecodeYAML[T any](data []byte) (T, error) {
	var zero T
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return zero, err
	}
	if len(document.Content) != 1 {
		return zero, fmt.Errorf("expected exactly one YAML document")
	}
	if err := rejectDuplicateKeys(document.Content[0], "$"); err != nil {
		return zero, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, err
	}
	return value, nil
}

func rejectDuplicateKeys(node *yaml.Node, path string) error {
	if node.Kind == yaml.MappingNode {
		seen := map[string]struct{}{}
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index].Value
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate key %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := rejectDuplicateKeys(node.Content[index+1], path+"."+key); err != nil {
				return err
			}
		}
		return nil
	}
	for index, child := range node.Content {
		if err := rejectDuplicateKeys(child, fmt.Sprintf("%s[%d]", path, index)); err != nil {
			return err
		}
	}
	return nil
}
