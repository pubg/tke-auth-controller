package internal

import (
	"github.com/pkg/errors"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
)


func ConfigsToCRBs(config v1.ConfigMap) (crb []*v12.ClusterRoleBinding, err error) {
	data := config.Data["bindings"]

	if data == "" {
		return nil, errors.Errorf("data of configMap is empty. configName: %s, namespace: %s\n", config.Name, config.Namespace)
	}

	var roleBindings map[string]TKEAuthClusterRoleBinding
	err = yaml.Unmarshal([]byte(data), roleBindings)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal bindings data, configName: %s, config: %s\n", config.Name, data)
	}

	converted := funk.Map(roleBindings, toClusterRoleBinding).([]*v12.ClusterRoleBinding)

	return converted, nil
}

func toClusterRoleBinding(key string, value TKEAuthClusterRoleBinding, ) *v12.ClusterRoleBinding {
	CNs := getCommonNames()

	subjects := funk.Map(value.Users, usersToSubjects).([]v12.Subject)
	apiGroup := "rbac.authorization.k8s.io"
	roleRef := v12.RoleRef{
		Kind: "ClusterRole",
		Name: value.RoleName,
		APIGroup: apiGroup,
	}

	crb := CreateClusterRoleBinding(key, roleRef, subjects)

	return crb
}

func usersToSubjects(users []string) []v12.Subject {
	subjects := make([]v12.Subject, 0)

	for _, user := range users {
		subjects = append(subjects, v12.Subject{
			Kind:      "User",
			APIGroup:  "rbac.authorization.k8s.io",
			Name:      user,
		})
	}

	return subjects
}

func getCommonNames(client tke.Client, clusterId string, names []*string) (CNs []*string, err error) {
	req := tke.NewDescribeClusterCommonNamesRequest()
	req.ClusterId = &clusterId
	req.SubaccountUins = names

	res, err := client.DescribeClusterCommonNames(req)
	if err != nil {
		return nil, err
	}

	CNs = funk.Map(res.Response.CommonNames, func (name tke.CommonName) *string { return name.CN }).([]*string)

	return CNs, nil
}
