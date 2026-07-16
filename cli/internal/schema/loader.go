package schema

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// LoadYAML decodes exactly one YAML document with duplicate and unknown fields rejected.
func LoadYAML[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", path, err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return zero, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(document.Content) != 1 {
		return zero, fmt.Errorf("decode %s: expected exactly one YAML document", path)
	}
	if err := rejectDuplicateKeys(document.Content[0], "$"); err != nil {
		return zero, fmt.Errorf("decode %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s: %w", path, err)
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
