// Package inline collapses references that resolve to a single item into the
// groups that reference them.
package inline

import "github.com/glynternet/packing/pkg/api"

// SingleItemGroups inlines references that resolve to a single item into the
// groups that reference them, dropping the standalone single-item group.
//
// A group is inlined when it has exactly one item, no refs, and is referenced
// by at least one other group. Its single item is appended to the items of
// every group that referenced it, and the reference is removed. Groups that are
// not referenced by anything are left as-is so their item is never lost.
//
// The transform is single-pass: the set of collapsible groups is computed from
// the input, so a group that only becomes single-item as a result of another
// group collapsing is not itself collapsed. Surviving groups keep their
// Requires unchanged.
func SingleItemGroups(groups []api.Group) []api.Group {
	referenced := make(map[string]bool)
	for _, g := range groups {
		for _, ref := range g.Contents.Refs {
			referenced[ref] = true
		}
	}

	// collapsible maps a single-item group's name to its one item.
	collapsible := make(map[string]string)
	for _, g := range groups {
		if len(g.Contents.Refs) == 0 && len(g.Contents.Items) == 1 && referenced[g.Name] {
			collapsible[g.Name] = g.Contents.Items[0]
		}
	}

	if len(collapsible) == 0 {
		return groups
	}

	out := make([]api.Group, 0, len(groups))
	for _, g := range groups {
		if _, ok := collapsible[g.Name]; ok {
			// Drop the standalone single-item group; it is inlined into its parents.
			continue
		}

		var refs []string
		var inlined []string
		for _, ref := range g.Contents.Refs {
			if item, ok := collapsible[ref]; ok {
				inlined = append(inlined, item)
			} else {
				refs = append(refs, ref)
			}
		}

		if len(inlined) > 0 {
			// Build a fresh items slice so the caller's input is never mutated.
			items := make([]string, 0, len(g.Contents.Items)+len(inlined))
			items = append(items, g.Contents.Items...)
			items = append(items, inlined...)
			g.Contents.Refs = refs
			g.Contents.Items = items
		}

		out = append(out, g)
	}
	return out
}
