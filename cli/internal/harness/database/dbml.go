package database

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Diff is an entity-level DBML change summary used before canonical writes.
type Diff struct {
	AddedTables        []string `json:"added_tables"`
	RemovedTables      []string `json:"removed_tables"`
	AddedColumns       []string `json:"added_columns"`
	RemovedColumns     []string `json:"removed_columns"`
	ChangedColumns     []string `json:"changed_columns"`
	AddedRelations     []string `json:"added_relations"`
	RemovedRelations   []string `json:"removed_relations"`
	AddedIndexes       []string `json:"added_indexes"`
	RemovedIndexes     []string `json:"removed_indexes"`
	AddedNotes         []string `json:"added_notes"`
	RemovedNotes       []string `json:"removed_notes"`
	AddedDefinitions   []string `json:"added_definitions"`
	RemovedDefinitions []string `json:"removed_definitions"`
	ChangedDefinitions []string `json:"changed_definitions"`
}

type dbmlModel struct {
	Tables      map[string]map[string]string
	Relations   map[string]struct{}
	Indexes     map[string]struct{}
	Notes       map[string]struct{}
	Definitions map[string]string
}

// SemanticDiff compares tables, columns, relationships, indexes, and single-line notes independent of formatting.
func SemanticDiff(before, after []byte) (Diff, error) {
	left, err := parseDBML(before)
	if err != nil {
		return Diff{}, err
	}
	right, err := parseDBML(after)
	if err != nil {
		return Diff{}, err
	}
	diff := Diff{
		AddedTables: differenceKeys(right.Tables, left.Tables), RemovedTables: differenceKeys(left.Tables, right.Tables),
		AddedColumns: []string{}, RemovedColumns: []string{}, ChangedColumns: []string{},
		AddedRelations: differenceSet(right.Relations, left.Relations), RemovedRelations: differenceSet(left.Relations, right.Relations),
		AddedIndexes: differenceSet(right.Indexes, left.Indexes), RemovedIndexes: differenceSet(left.Indexes, right.Indexes),
		AddedNotes: differenceSet(right.Notes, left.Notes), RemovedNotes: differenceSet(left.Notes, right.Notes),
		AddedDefinitions: differenceStringKeys(right.Definitions, left.Definitions), RemovedDefinitions: differenceStringKeys(left.Definitions, right.Definitions),
		ChangedDefinitions: []string{},
	}
	for key, signature := range right.Definitions {
		if previous, exists := left.Definitions[key]; exists && previous != signature {
			diff.ChangedDefinitions = append(diff.ChangedDefinitions, key)
		}
	}
	for table, columns := range right.Tables {
		if previous, exists := left.Tables[table]; exists {
			for _, column := range differenceColumnKeys(columns, previous) {
				diff.AddedColumns = append(diff.AddedColumns, table+"."+column)
			}
			for column, signature := range columns {
				if oldSignature, exists := previous[column]; exists && oldSignature != signature {
					diff.ChangedColumns = append(diff.ChangedColumns, table+"."+column)
				}
			}
		}
	}
	for table, columns := range left.Tables {
		if current, exists := right.Tables[table]; exists {
			for _, column := range differenceColumnKeys(columns, current) {
				diff.RemovedColumns = append(diff.RemovedColumns, table+"."+column)
			}
		}
	}
	sort.Strings(diff.AddedColumns)
	sort.Strings(diff.RemovedColumns)
	sort.Strings(diff.ChangedColumns)
	sort.Strings(diff.ChangedDefinitions)
	return diff, nil
}

func parseDBML(data []byte) (dbmlModel, error) {
	model := dbmlModel{Tables: map[string]map[string]string{}, Relations: map[string]struct{}{}, Indexes: map[string]struct{}{}, Notes: map[string]struct{}{}, Definitions: map[string]string{}}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), maxDBMLEntryBytes)
	current := ""
	inIndexes := false
	blockKey := ""
	blockKind := ""
	blockDepth := 0
	blockLines := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(stripDBMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		if blockDepth > 0 {
			blockLines = append(blockLines, line)
			blockDepth += braceDelta(line)
			if blockDepth < 0 {
				return model, fmt.Errorf("unmatched DBML closing brace in %s", blockKey)
			}
			if blockDepth == 0 {
				signature := normalizeDBMLBlock(blockKey, blockLines)
				if blockKind == "ref" {
					model.Relations[signature] = struct{}{}
				} else {
					model.Definitions[blockKey] = signature
				}
				blockKey, blockKind, blockLines = "", "", nil
			}
			continue
		}
		if current == "" {
			if key, kind, ok := topLevelBlock(line); ok {
				if kind != "ref" {
					if _, duplicate := model.Definitions[key]; duplicate {
						return model, fmt.Errorf("duplicate DBML definition %s", key)
					}
				}
				blockKey, blockKind = key, kind
				blockLines = []string{line}
				blockDepth = braceDelta(line)
				if blockDepth <= 0 {
					if blockDepth < 0 {
						return model, fmt.Errorf("unmatched DBML closing brace in %s", key)
					}
					signature := normalizeDBMLBlock(blockKey, blockLines)
					if kind == "ref" {
						model.Relations[signature] = struct{}{}
					} else {
						model.Definitions[key] = signature
					}
					blockKey, blockKind, blockLines = "", "", nil
				}
				continue
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "table ") {
			if current != "" {
				return model, fmt.Errorf("nested DBML table declaration is invalid")
			}
			fields := strings.Fields(line)
			if len(fields) < 2 || !strings.Contains(line, "{") {
				return model, fmt.Errorf("DBML table name is required")
			}
			name := fields[1]
			name = strings.Trim(name, "`\"")
			if name == "" {
				return model, fmt.Errorf("DBML table name is required")
			}
			if _, exists := model.Tables[name]; exists {
				return model, fmt.Errorf("duplicate DBML table %s", name)
			}
			model.Tables[name] = map[string]string{}
			inner := strings.TrimSpace(strings.SplitN(line, "{", 2)[1])
			if strings.Contains(inner, "}") {
				beforeClose, afterClose, _ := strings.Cut(inner, "}")
				if strings.TrimSpace(afterClose) != "" || strings.Contains(beforeClose, "{") {
					return model, fmt.Errorf("invalid inline DBML table %s", name)
				}
				if strings.TrimSpace(beforeClose) != "" {
					if err := addDBMLColumn(model, name, strings.TrimSpace(beforeClose)); err != nil {
						return model, err
					}
				}
				current = ""
			} else {
				current = name
			}
			continue
		}
		if line == "}" {
			if inIndexes {
				inIndexes = false
				continue
			}
			if current == "" {
				return model, fmt.Errorf("unmatched DBML closing brace")
			}
			current = ""
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "ref:") || strings.HasPrefix(lower, "ref ") {
			model.Relations[normalizeDBMLLine(line)] = struct{}{}
			continue
		}
		if current != "" && strings.HasPrefix(lower, "indexes") && strings.Contains(line, "{") {
			inIndexes = true
			continue
		}
		if current != "" && inIndexes {
			model.Indexes[current+"."+normalizeDBMLLine(line)] = struct{}{}
			continue
		}
		if strings.HasPrefix(lower, "note:") {
			scope := "global"
			if current != "" {
				scope = current
			}
			note := strings.TrimSpace(line[len("Note:"):])
			model.Notes[scope+"."+normalizeDBMLLine(note)] = struct{}{}
			continue
		}
		if current != "" && line != "{" {
			if err := addDBMLColumn(model, current, line); err != nil {
				return model, err
			}
			continue
		}
		return model, fmt.Errorf("DBML content appears outside a table: %s", line)
	}
	if err := scanner.Err(); err != nil {
		return model, err
	}
	if current != "" || inIndexes || blockDepth != 0 {
		return model, fmt.Errorf("DBML table or indexes block is not closed")
	}
	return model, nil
}

func topLevelBlock(line string) (string, string, bool) {
	if !strings.Contains(line, "{") {
		return "", "", false
	}
	header := normalizeDBMLLine(strings.TrimSpace(strings.SplitN(line, "{", 2)[0]))
	fields := strings.Fields(header)
	if len(fields) < 2 {
		return "", "", false
	}
	kind := strings.ToLower(fields[0])
	switch kind {
	case "project", "enum", "tablegroup", "tablepartial", "diagramview", "note", "ref":
	default:
		return "", "", false
	}
	name := strings.Trim(fields[1], "`\"")
	if name == "" {
		return "", "", false
	}
	return kind + " " + name, kind, true
}

func normalizeDBMLBlock(key string, lines []string) string {
	normalized := make([]string, 0, len(lines))
	for index, line := range lines {
		if index == 0 {
			_, suffix, found := strings.Cut(line, "{")
			if found {
				line = key + " {" + suffix
			}
		}
		normalized = append(normalized, normalizeDBMLLine(line))
	}
	return strings.Join(normalized, " ")
}

func stripDBMLComment(line string) string {
	quote := rune(0)
	escaped := false
	runes := []rune(line)
	for index, value := range runes {
		if escaped {
			escaped = false
			continue
		}
		if value == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if value == quote {
				quote = 0
			}
			continue
		}
		if value == '\'' || value == '"' || value == '`' {
			quote = value
			continue
		}
		if value == '/' && index+1 < len(runes) && runes[index+1] == '/' {
			return string(runes[:index])
		}
	}
	return line
}

func braceDelta(line string) int {
	delta := 0
	quote := rune(0)
	escaped := false
	for _, value := range []rune(line) {
		if escaped {
			escaped = false
			continue
		}
		if value == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if value == quote {
				quote = 0
			}
			continue
		}
		if value == '\'' || value == '"' || value == '`' {
			quote = value
			continue
		}
		switch value {
		case '{':
			delta++
		case '}':
			delta--
		}
	}
	return delta
}

func addDBMLColumn(model dbmlModel, table, line string) error {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return fmt.Errorf("DBML column in %s requires a name and type", table)
	}
	name := strings.Trim(fields[0], "`\"")
	if name == "" {
		return fmt.Errorf("DBML column name is required")
	}
	if _, duplicate := model.Tables[table][name]; duplicate {
		return fmt.Errorf("duplicate DBML column %s.%s", table, name)
	}
	model.Tables[table][name] = normalizeDBMLLine(strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
	if relation, ok := inlineRelation(line); ok {
		model.Relations[table+"."+name+" "+relation] = struct{}{}
	}
	return nil
}

func normalizeDBMLLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func inlineRelation(line string) (string, bool) {
	lower := strings.ToLower(line)
	index := strings.Index(lower, "ref:")
	if index < 0 {
		return "", false
	}
	value := strings.TrimSpace(line[index+len("ref:"):])
	if end := strings.IndexAny(value, ",]"); end >= 0 {
		value = value[:end]
	}
	value = normalizeDBMLLine(value)
	return value, value != ""
}

func differenceKeys(left, right map[string]map[string]string) []string {
	result := []string{}
	for key := range left {
		if _, exists := right[key]; !exists {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func differenceColumnKeys(left, right map[string]string) []string {
	result := []string{}
	for key := range left {
		if _, exists := right[key]; !exists {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func differenceStringKeys(left, right map[string]string) []string {
	result := []string{}
	for key := range left {
		if _, exists := right[key]; !exists {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func differenceSet(left, right map[string]struct{}) []string {
	result := []string{}
	for key := range left {
		if _, exists := right[key]; !exists {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
