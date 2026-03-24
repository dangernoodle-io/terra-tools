package rename

import (
	"sort"

	tfjson "github.com/hashicorp/terraform-json"
)

// RenamePair represents a resource address rename.
type RenamePair struct {
	From string
	To   string
}

// typeNameKey groups resources by both type and name.
type typeNameKey struct {
	Type string
	Name string
}

// Candidate represents a destroyed resource with potential create matches of the same type and name.
type Candidate struct {
	Destroy *tfjson.ResourceChange
	Creates []*tfjson.ResourceChange
}

// DetectFromPlan returns all resource changes where PreviousAddress is set.
// These are definite renames — Terraform already knows the resource moved.
func DetectFromPlan(p *tfjson.Plan) []RenamePair {
	var pairs []RenamePair
	for _, rc := range p.ResourceChanges {
		if rc.PreviousAddress != "" {
			pairs = append(pairs, RenamePair{
				From: rc.PreviousAddress,
				To:   rc.Address,
			})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].From < pairs[j].From
	})
	return pairs
}

// MatchDestroyCreate finds destroy/create pairs of the same resource type and name as
// rename candidates. It excludes resources that already have PreviousAddress set
// (those are definite renames handled by DetectFromPlan).
func MatchDestroyCreate(p *tfjson.Plan) []Candidate {
	// Build set of addresses with PreviousAddress (already detected as renames).
	knownMoved := make(map[string]bool)
	for _, rc := range p.ResourceChanges {
		if rc.PreviousAddress != "" {
			knownMoved[rc.Address] = true
			knownMoved[rc.PreviousAddress] = true
		}
	}

	// Group destroys and creates by type and name.
	destroysByKey := make(map[typeNameKey][]*tfjson.ResourceChange)
	createsByKey := make(map[typeNameKey][]*tfjson.ResourceChange)

	for _, rc := range p.ResourceChanges {
		if rc.Change == nil || knownMoved[rc.Address] {
			continue
		}
		key := typeNameKey{Type: rc.Type, Name: rc.Name}
		if rc.Change.Actions.Delete() {
			destroysByKey[key] = append(destroysByKey[key], rc)
		}
		if rc.Change.Actions.Create() {
			createsByKey[key] = append(createsByKey[key], rc)
		}
	}

	// Build candidates: for each destroy, find creates of the same type and name.
	var candidates []Candidate
	for key, destroys := range destroysByKey {
		creates, ok := createsByKey[key]
		if !ok || len(creates) == 0 {
			continue
		}
		for _, d := range destroys {
			candidates = append(candidates, Candidate{
				Destroy: d,
				Creates: creates,
			})
		}
	}

	// Sort for deterministic output.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Destroy.Address < candidates[j].Destroy.Address
	})

	return candidates
}
