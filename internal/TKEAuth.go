package internal

import (
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/rbac/v1"
	v15 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type User struct {
	ValueType string `yaml:"type"`
	Value string `yaml:"value"`
}

type TKEAuth struct {
	BindingName string   `yaml:"bindingName"`
	RoleName    string   `yaml:"roleName"`
	Users       []User `yaml:"users"`
}

func (t *TKEAuth) ToClusterRoleBinding() *v1.ClusterRoleBinding {
	roleRef := toClusterRoleRef(t.BindingName)
	subjects := funk.Map(t.Users, userToSubject).([]v1.Subject)

	crb := &v1.ClusterRoleBinding{
		TypeMeta: v15.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: v15.ObjectMeta{
			Name:                       t.BindingName,
		},
		Subjects:   subjects,
		RoleRef:    roleRef,
	}

	return crb
}

func toClusterRoleRef(roleName string) v1.RoleRef {
	ref := v1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     roleName,
	}

	return ref
}

func userToSubject(user string) v1.Subject {
	subject := v1.Subject{
		Kind:      "User",
		APIGroup:  "rbac.authorization.k8s.io",
		Name:      user,
	}

	return subject
}
