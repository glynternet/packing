package graph

import api "github.com/glynternet/packing/pkg/api/build"

func From(gs []*api.Group) []Group {
	ig := groups(gs).importGraph()
	var ggs []Group
	for _, g := range gs {
		ggs = append(ggs, Group{
			Group:      g,
			ImportedBy: ig[g.Name],
		})
	}
	return ggs
}

// importGraph is a Group name as key and all groups that import it as the value
type importGraph map[string][]string

type groups []*api.Group

func (gs groups) thatImport(key string) []string {
	var groups []string
	for _, g := range gs {
		for _, gk := range g.Contents.GroupKeys {
			if gk.Key == key {
				groups = append(groups, g.Name)
			}
		}
	}
	return groups
}

func (gs groups) importGraph() importGraph {
	graph := make(importGraph)
	for _, g := range gs {
		graph[g.Name] = gs.thatImport(g.Name)
	}
	return graph
}
