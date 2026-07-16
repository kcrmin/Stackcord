package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"fullstack-orchestrator/cli/internal/domain"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed definitions/*.json
var definitions embed.FS

var (
	compiledSchemas sync.Map
	secretName      = regexp.MustCompile(`(?i)(^|[_-])(token|password|secret|private[_-]?key)($|[_-])`)
)

// Validate applies the registered JSON Schema and repository-wide secret-name policy.
func Validate(kind string, value any) []domain.Item {
	filename := "definitions/" + kind + ".schema.json"
	if _, err := definitions.ReadFile(filename); err != nil {
		return []domain.Item{{Code: "schema.unknown-kind", Message: fmt.Sprintf("schema kind %q is not registered", kind)}}
	}

	if paths := secretPaths(value, "$"); len(paths) > 0 {
		sort.Strings(paths)
		return []domain.Item{{Code: "schema.invalid", Message: "secret-like properties are prohibited", Refs: paths}}
	}

	schemaValue, err := compiled(kind, filename)
	if err != nil {
		return []domain.Item{{Code: "schema.internal", Message: err.Error()}}
	}
	normalized, err := normalizeJSON(value)
	if err != nil {
		return []domain.Item{{Code: "schema.invalid", Message: err.Error()}}
	}
	if err := schemaValue.Validate(normalized); err != nil {
		return []domain.Item{{Code: "schema.invalid", Message: err.Error()}}
	}
	return nil
}

func compiled(kind, filename string) (*jsonschema.Schema, error) {
	if cached, ok := compiledSchemas.Load(kind); ok {
		return cached.(*jsonschema.Schema), nil
	}
	data, err := definitions.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse embedded %s schema: %w", kind, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	location := "https://orchestrator.invalid/schemas/" + kind + ".schema.json"
	if err := compiler.AddResource(location, document); err != nil {
		return nil, fmt.Errorf("register %s schema: %w", kind, err)
	}
	result, err := compiler.Compile(location)
	if err != nil {
		return nil, fmt.Errorf("compile %s schema: %w", kind, err)
	}
	compiledSchemas.Store(kind, result)
	return result, nil
}

func normalizeJSON(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode schema input: %w", err)
	}
	var normalized any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("decode schema input: %w", err)
	}
	return normalized, nil
}

func secretPaths(value any, path string) []string {
	var result []string
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childPath := path + "." + key
			if secretName.MatchString(key) {
				result = append(result, childPath)
			}
			result = append(result, secretPaths(child, childPath)...)
		}
	case []any:
		for index, child := range typed {
			result = append(result, secretPaths(child, fmt.Sprintf("%s[%d]", path, index))...)
		}
	}
	return result
}
