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
	AddedRelations   []string `json:"added_relations"`
	RemovedRelations []string `json:"removed_relations"`
}

type dbmlModel struct {
	Tables    map[string]map[string]struct{}
	Relations map[string]struct{}
}

// SemanticDiff compares tables, columns, and relationships independent of formatting.
func SemanticDiff(before, after []byte) (Diff, error) {
	left, err := parseDBML(before)
	if err != nil {
		return Diff{}, err
	}
	right, err := parseDBML(after)
	if err != nil {
		return Diff{}, err
	}
	diff := Diff{AddedTables: differenceKeys(right.Tables, left.Tables), RemovedTables: differenceKeys(left.Tables, right.Tables), AddedRelations: differenceSet(right.Relations, left.Relations), RemovedRelations: differenceSet(left.Relations, right.Relations)}
	for table, columns := range right.Tables {
		if previous, exists := left.Tables[table]; exists {
			for _, column := range differenceSet(columns, previous) {
				diff.AddedColumns = append(diff.AddedColumns, table+"."+column)
			}
		}
	}
	for table, columns := range left.Tables {
		if current, exists := right.Tables[table]; exists {
			for _, column := range differenceSet(columns, current) {
				diff.RemovedColumns = append(diff.RemovedColumns, table+"."+column)
			}
		}
	}
	sort.Strings(diff.AddedColumns)
	sort.Strings(diff.RemovedColumns)
	return diff, nil
}

func parseDBML(data []byte) (dbmlModel, error) {
	model := dbmlModel{Tables: map[string]map[string]struct{}{}, Relations: map[string]struct{}{}}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	current := ""
	for scanner.Scan() {
		line := strings.TrimSpace(strings.Split(scanner.Text(), "//")[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "table ") {
			name := strings.Fields(line)[1]
			name = strings.Trim(name, "`\"")
			if name == "" {
				return model, fmt.Errorf("DBML table name is required")
			}
			current = name
			model.Tables[name] = map[string]struct{}{}
			continue
		}
		if line == "}" {
			current = ""
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "ref:") {
			model.Relations[strings.Join(strings.Fields(line), " ")] = struct{}{}
			continue
		}
		if current != "" && line != "{" {
			name := strings.Trim(strings.Fields(line)[0], "`\"")
			model.Tables[current][name] = struct{}{}
		}
	}
	return model, scanner.Err()
}

func differenceKeys(left, right map[string]map[string]struct{}) []string {
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
