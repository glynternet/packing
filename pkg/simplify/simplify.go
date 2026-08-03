// Package simplify minifies a selection by removing top-level refs and items
// that are already covered by other selected refs.
//
// A selection is the seed api.Contents a user edits (a CLI selection file or the
// web UI selection editor). Hand-written selections often list a group at the top
// level even though an umbrella group already references it, and list an item at
// the top level even though a selected group already contains it. Simplify
// removes those redundant entries, given the full expansion of the selection.
//
// Removing a redundant ref is output-neutral: the loaded group set is the
// transitive closure of the seed refs, so dropping a ref whose group is already
// reachable from another kept ref does not change that closure. Removing a
// redundant item drops a genuine duplicate list entry (the item showed under both
// the synthetic "Individual Items" group and the real group that contains it).
//
// Requires (req:) are left untouched: the loader does not currently expand
// requirements, so they contribute nothing to reachability.
package simplify

import (
	"strings"

	"github.com/glynternet/packing/pkg/api"
)

// refTag mirrors the unexported list.referencePrefix: the tag naming a reference
// line in the selection grammar (see pkg/list/stringprocessor.go).
const refTag = "ref"

// Result holds the kept and removed refs/items of a simplified selection.
type Result struct {
	Refs         []string // kept top-level refs, first-seen order
	Items        []string // kept top-level items, first-seen order
	RemovedRefs  []string
	RemovedItems []string
}

// Simplify computes which of the seed's top-level refs and items are redundant
// given the full expansion groups (each with its Refs and Items). A ref is
// redundant when its group is reachable from another of the (kept) seed refs; an
// item is redundant when it appears in some group reachable from the seed refs.
func Simplify(seed api.Contents, groups []api.Group) Result {
	groupRefs := make(map[string][]string, len(groups))
	groupItems := make(map[string][]string, len(groups))
	for _, g := range groups {
		groupRefs[g.Name] = g.Contents.Refs
		groupItems[g.Name] = g.Contents.Items
	}

	// closureFrom returns the set of group names reachable by following refs from
	// the given start names (the start names themselves included). The synthetic
	// "Individual Items" group is never reached because nothing references it, so
	// it self-excludes from item redundancy.
	closureFrom := func(start []string) map[string]bool {
		seen := make(map[string]bool)
		stack := append([]string(nil), start...)
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[n] {
				continue
			}
			seen[n] = true
			stack = append(stack, groupRefs[n]...)
		}
		return seen
	}

	// Redundant refs: greedy and cycle-safe. Re-evaluate each candidate against
	// the current working set (not the original), so a mutual A<->B cycle removes
	// exactly one of the pair — the second check sees the first already gone.
	working := dedupe(seed.Refs)
	var removedRefs []string
	for i := 0; i < len(working); {
		r := working[i]
		rest := make([]string, 0, len(working)-1)
		rest = append(rest, working[:i]...)
		rest = append(rest, working[i+1:]...)
		if closureFrom(rest)[r] {
			removedRefs = append(removedRefs, r)
			working = rest // drop r; working[i] is now the next element
			continue
		}
		i++
	}

	// Redundant items: any item already present in a group reachable from the
	// seed refs (closure is identical whether computed from the kept or original
	// refs, since ref-simplification is closure-preserving).
	reachableItems := make(map[string]bool)
	for name := range closureFrom(seed.Refs) {
		for _, it := range groupItems[name] {
			reachableItems[it] = true
		}
	}
	var keptItems, removedItems []string
	for _, it := range dedupe(seed.Items) {
		if reachableItems[it] {
			removedItems = append(removedItems, it)
		} else {
			keptItems = append(keptItems, it)
		}
	}

	return Result{
		Refs:         working,
		Items:        keptItems,
		RemovedRefs:  removedRefs,
		RemovedItems: removedItems,
	}
}

// Filter returns text with the ref/item lines whose value is in removedRefs or
// removedItems dropped, preserving comments, blank lines, ordering, and every
// other line verbatim. Lines are classified exactly as list.ParseContentsDefinition
// classifies them (strip # comment, trim, split on the first ':'); req: lines,
// comments and blanks are never dropped.
func Filter(text string, removedRefs, removedItems []string) string {
	dropRef := toSet(removedRefs)
	dropItem := toSet(removedItems)

	lines := strings.Split(text, "\n")
	out := lines[:0:0]
	for _, line := range lines {
		if dropLine(line, dropRef, dropItem) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// dropLine reports whether line is a ref/item line whose value is in the given
// removal sets. Classification mirrors pkg/list/stringprocessor.go.
func dropLine(line string, dropRef, dropItem map[string]bool) bool {
	s := line
	if i := strings.IndexRune(s, '#'); i != -1 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		// Blank or comment-only line: always kept.
		return false
	}
	if i := strings.IndexRune(s, ':'); i != -1 {
		if s[:i] == refTag {
			return dropRef[strings.TrimSpace(s[i+1:])]
		}
		// req: or any other tag: kept.
		return false
	}
	// No colon: an item line.
	return dropItem[s]
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func toSet(in []string) map[string]bool {
	set := make(map[string]bool, len(in))
	for _, s := range in {
		set[s] = true
	}
	return set
}
