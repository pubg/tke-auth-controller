package internal_test

import (
	"example.com/tke-auth-controller/internal"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/rbac/v1"
	"testing"
)

func TestCreateClusterRoleBinding(t *testing.T) {
	name := "foobar"
	crb := internal.CreateClusterRoleBinding(name, v1.RoleRef{}, make([]v1.Subject, 0))

	assert.Equal(t, crb.Name, name)
	assert.Equal(t, crb.Kind, "ClusterRoleBinding")
}