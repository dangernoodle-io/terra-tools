package plan

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestFilterCreates(t *testing.T) {
	tests := []struct {
		name     string
		plan     *tfjson.Plan
		expected int
	}{
		{
			name: "only creates",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.one", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}}},
					{Address: "acme_widget.two", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}}},
				},
			},
			expected: 2,
		},
		{
			name: "mixed actions",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.create", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}}},
					{Address: "acme_widget.delete", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}}},
					{Address: "acme_widget.update", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}}},
				},
			},
			expected: 1,
		},
		{
			name:     "empty plan",
			plan:     &tfjson.Plan{},
			expected: 0,
		},
		{
			name: "nil change skipped",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.nil", Change: nil},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterCreates(tt.plan)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestFilterDestroys(t *testing.T) {
	tests := []struct {
		name     string
		plan     *tfjson.Plan
		expected int
	}{
		{
			name: "only deletes",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.one", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}}},
					{Address: "acme_widget.two", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}}},
				},
			},
			expected: 2,
		},
		{
			name: "mixed actions",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.create", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}}},
					{Address: "acme_widget.delete", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}}},
					{Address: "acme_widget.update", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}}},
				},
			},
			expected: 1,
		},
		{
			name:     "empty plan",
			plan:     &tfjson.Plan{},
			expected: 0,
		},
		{
			name: "nil change skipped",
			plan: &tfjson.Plan{
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: "acme_widget.nil", Change: nil},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterDestroys(tt.plan)
			assert.Len(t, result, tt.expected)
		})
	}
}
