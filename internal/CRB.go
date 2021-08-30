package internal

import v1 "k8s.io/api/rbac/v1"

func CreateClusterRoleBinding(name string, roleRef v1.RoleRef, subjects []v1.Subject) *v1.ClusterRoleBinding {
	crb := new(v1.ClusterRoleBinding)
	crb.Name = name
	crb.Kind = "ClusterRoleBinding"
	crb.RoleRef = roleRef
	crb.Subjects = subjects

	return crb
}
