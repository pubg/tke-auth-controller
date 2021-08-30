package internal_test

import (
	"github.com/stretchr/testify/assert"
	v12 "k8s.io/api/rbac/v1"
	"testing"
)

func TestClusterRoleBindingCreation(t *testing.T) {
	crb := v12.ClusterRoleBinding{}

	assert.Equal(t, crb.Kind, "ClusterRoleBinding", "Kind should be clusterRoleBinding")
}