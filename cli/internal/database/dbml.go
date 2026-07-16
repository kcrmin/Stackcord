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
	AddedTables      []string `json:"added_tables"`
	RemovedTables    []string `json:"removed_tables"`
	AddedColumns     []string `json:"added_columns"`
	RemovedColumns   []string `json:"removed_columns"`
	ChangedColumns   []string `json:"changed_columns"`
	AddedRelations   []string `json:"added_relations"`
	RemovedRelations []string `json:"removed_relations"`
	AddedIndexes     []string `json:"added_indexes"`
	RemovedIndexes   []string `json:"removed_indexes"`
	AddedNotes       []string `json:"added_notes"`
	RemovedNotes     []string `json:"removed_notes"`
}

type dbmlModel struct {
	Tables    map[string]map[string]string
	Relations map[string]struct{}
	Indexes   map[string]struct{}
	Notes     map[string]struct{}
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
		AddedRelations: differenceSet(right.Relations, left.Relations), RemovedRelations: differenceSet(left.Relations, right.Relations),
		AddedIndexes: differenceSet(right.Indexes, left.Indexes), RemovedIndexes: differenceSet(left.Indexes, right.Indexes),
		AddedNotes: differenceSet(right.Notes, left.Notes), RemovedNotes: differenceSet(left.Notes, right.Notes),
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
	return diff, nil
}

func parseDBML(data []byte) (dbmlModel, error) {
	model := dbmlModel{Tables: map[string]map[string]string{}, Relations: map[string]struct{}{}, Indexes: map[string]struct{}{}, Notes: map[string]struct{}{}}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	current := ""
	inIndexes := false
	for scanner.Scan() {
		line := strings.TrimSpace(strings.Split(scanner.Text(), "//")[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "table ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return model, fmt.Errorf("DBML table name is required")
			}
			name := fields[1]
			name = strings.Trim(name, "`\"")
			if name == "" {
				return model, fmt.Errorf("DBML table name is required")
			}
			current = name
			model.Tables[name] = map[string]string{}
			continue
		}
		if line == "}" {
			if inIndexes {
				inIndexes = false
				continue
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
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			name := strings.Trim(fields[0], "`\"")
			model.Tables[current][name] = normalizeDBMLLine(strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
			if relation, ok := inlineRelation(line); ok {
				model.Relations[current+"."+name+" "+relation] = struct{}{}
			}
		}
	}
	return model, scanner.Err()
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
