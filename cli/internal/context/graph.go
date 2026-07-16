package context

import "sort"

func buildImpact(index map[string]IndexEntry) map[string][]string {
	sets := map[string]map[string]struct{}{}
	for id, entry := range index {
		for _, dependency := range entry.Refs {
			addEdge(sets, dependency, id)
		}
		for _, source := range entry.Sources {
			addEdge(sets, source.Source, id)
		}
		if _, exists := sets[id]; !exists {
			sets[id] = map[string]struct{}{}
		}
	}
	result := make(map[string][]string, len(sets))
	for source, targets := range sets {
		for target := range targets {
			result[source] = append(result[source], target)
		}
		sort.Strings(result[source])
	}
	return result
}

func addEdge(graph map[string]map[string]struct{}, source, target string) {
	if source == "" || target == "" || source == target {
		return
	}
	if graph[source] == nil {
		graph[source] = map[string]struct{}{}
	}
	graph[source][target] = struct{}{}
}

func propagateStale(seeds map[string]struct{}, impact map[string][]string) []string {
	queue := make([]string, 0, len(seeds))
	seen := make(map[string]struct{}, len(seeds))
	for seed := range seeds {
		queue = append(queue, seed)
		seen[seed] = struct{}{}
	}
	sort.Strings(queue)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range impact[current] {
			if _, exists := seen[next]; exists {
				continue
			}
			seen[next] = struct{}{}
			queue = append(queue, next)
		}
		sort.Strings(queue)
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
