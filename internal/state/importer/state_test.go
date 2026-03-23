package importer

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

func TestCollectAddresses_FlatModule(t *testing.T) {
	mod := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_s3_bucket.a"},
			{Address: "aws_iam_role.b"},
		},
	}

	addrs := collectAddresses(mod)
	assert.Equal(t, []string{"aws_s3_bucket.a", "aws_iam_role.b"}, addrs)
}

func TestCollectAddresses_NestedModules(t *testing.T) {
	child := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_iam_policy.child_policy"},
		},
	}

	parent := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_s3_bucket.parent_bucket"},
		},
		ChildModules: []*tfjson.StateModule{child},
	}

	addrs := collectAddresses(parent)
	assert.ElementsMatch(t, []string{"aws_s3_bucket.parent_bucket", "aws_iam_policy.child_policy"}, addrs)
}

func TestCollectAddresses_NilModule(t *testing.T) {
	addrs := collectAddresses(nil)
	assert.Nil(t, addrs)
}

func TestCollectAddresses_EmptyResources(t *testing.T) {
	mod := &tfjson.StateModule{}

	addrs := collectAddresses(mod)
	assert.Nil(t, addrs)
}

func TestCollectAddresses_EmptyResourcesWithChildren(t *testing.T) {
	child := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_vpc.child"},
		},
	}

	parent := &tfjson.StateModule{
		Resources:    []*tfjson.StateResource{},
		ChildModules: []*tfjson.StateModule{child},
	}

	addrs := collectAddresses(parent)
	assert.Equal(t, []string{"aws_vpc.child"}, addrs)
}
