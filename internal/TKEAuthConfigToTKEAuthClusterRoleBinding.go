package internal

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	v15 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// helper struct while converting TKEAuthConfig to TKEAuthClusterRoleBinding
type conversionHelper struct {
	configMap *v1.ConfigMap
	roleRef   *v12.RoleRef
	subjects  map[string]v12.Subject
}

// TKEAuthConfigsToAuthClusterRoleBindings creates ClusterRoleBindings from multiple TKEAuthConfig (role -> 1:N <- User)
func TKEAuthConfigsToAuthClusterRoleBindings(configs []*TKEAuthConfig) ([]*v12.ClusterRoleBinding, error) {
	// Map<RoleRefStr, Set<UserRef>>
	helpers := make(map[string]*conversionHelper)

	// creates 1:N clusterRoleBinding while removing duplicates
	for _, cfg := range configs {
		for _, binding := range cfg.GetBindings() {
			v, _ := helpers[binding.RoleName]
			if v == nil {
				v = &conversionHelper{
					configMap: cfg.configMap,
					roleRef: &v12.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     binding.RoleName,
					},
					subjects: make(map[string]v12.Subject),
				}
				helpers[binding.RoleName] = v
			}

			for _, user := range binding.Users {
				v.subjects[user] = v12.Subject{
					Kind:     "User",
					APIGroup: "rbac.authorization.k8s.io",
					Name:     user,
				}
			}
		}
	}

	CRBs := make([]*v12.ClusterRoleBinding, 0)

	for _, helper := range helpers {
		subjects := make([]v12.Subject, len(helper.subjects))

		for _, subject := range helper.subjects {
			subjects = append(subjects, subject)
		}

		crb := &v12.ClusterRoleBinding{
			TypeMeta: v15.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: v15.ObjectMeta{
				Name: helper.roleRef.Name,
				Annotations: map[string]string{ // TODO: make managing annotations externally
					AnnotationKeyManagedTKEAuthCRB: AnnotationValueManagedTKEAuthCRB,
				},
				OwnerReferences: []v15.OwnerReference{
					{
						APIVersion: helper.configMap.APIVersion,
						Kind:       helper.configMap.Kind,
						Name:       helper.configMap.Name,
						UID:        helper.configMap.UID,
					},
				},
				Finalizers:    nil,
				ClusterName:   "",
				ManagedFields: nil,
			},
			Subjects: subjects,
			RoleRef:  *helper.roleRef,
		}

		CRBs = append(CRBs, crb)
	}

	return CRBs, nil
}
