// Package inline collapses references that resolve to a single member into the
// groups that reference them. PassthroughRefs (single-ref chains) and
// SingleItemGroups (single-item groups) together implement "singleton"
// inlining: PassthroughRefs runs first and rewrites refs down to their
// terminal, so a chain ending in a single-item group is then inlined by
// SingleItemGroups.
package inline

import "github.com/glynternet/packing/pkg/api"

// PassthroughRefs collapses chains of "passthrough" groups. A passthrough group
// is one whose entire content is a single reference: exactly one ref, no items,
// and no requires. References to a passthrough head are rewritten to point at
// the chain's terminal — the first name reached that is not itself a
// passthrough — and the resolvable passthrough groups are dropped.
//
// The chain is followed with a visited set. If following the passthrough map
// revisits a name (a cycle) before reaching a non-passthrough terminal, that
// head is left unresolved: refs to it are not rewritten and it is not dropped,
// so cycles can never loop or lose data.
//
// Rewritten refs are deduplicated preserving first-seen order, because multiple
// heads can now point at a single terminal. Surviving groups keep their Items
// and Requires unchanged. The caller's input slices are never mutated; fresh
// slices are built for any rewritten refs.
//
// This is intended to run before SingleItemGroups so the two compose: a chain
// that terminates in a single-item group first has its refs rewritten to the
// terminal, which SingleItemGroups then inlines into its parents.
func PassthroughRefs(groups []api.Group) []api.Group {
	// passthrough maps a passthrough group's name to its single ref.
	passthrough := make(map[string]string)
	for _, g := range groups {
		c := g.Contents
		if len(c.Refs) == 1 && len(c.Items) == 0 && len(c.Requires) == 0 {
			passthrough[g.Name] = c.Refs[0]
		}
	}

	if len(passthrough) == 0 {
		return groups
	}

	// resolve returns the terminal for a passthrough head. ok is true only when
	// name is a passthrough whose chain reaches a non-passthrough terminal; a
	// name that is not a passthrough, or a chain that cycles, returns ok=false.
	resolve := func(name string) (string, bool) {
		if _, ok := passthrough[name]; !ok {
			return "", false
		}
		visited := make(map[string]bool)
		current := name
		for {
			next, isPassthrough := passthrough[current]
			if !isPassthrough {
				return current, true
			}
			if visited[current] {
				// Cycle detected before reaching a non-passthrough terminal.
				return "", false
			}
			visited[current] = true
			current = next
		}
	}

	out := make([]api.Group, 0, len(groups))
	for _, g := range groups {
		// Drop groups that are resolvable passthroughs; they are inlined into
		// whatever referenced them. Cycle passthroughs (unresolved) are kept.
		if _, resolved := resolve(g.Name); resolved {
			continue
		}

		if len(g.Contents.Refs) > 0 {
			// Build a fresh refs slice so the caller's input is never mutated.
			refs := make([]string, 0, len(g.Contents.Refs))
			seen := make(map[string]bool)
			for _, ref := range g.Contents.Refs {
				target := ref
				if terminal, resolved := resolve(ref); resolved {
					target = terminal
				}
				if seen[target] {
					continue
				}
				seen[target] = true
				refs = append(refs, target)
			}
			g.Contents.Refs = refs
		}

		out = append(out, g)
	}
	return out
}

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
