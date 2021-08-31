package internal

import (
	"gopkg.in/yaml.v3"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/informers/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	DataKeyBindingName = "bindingName"
	DataKeyRoleName = "roleName"
	DataKeyUsers = "users"
	AnnotationKeyTKEAuthConfigMap = "tke-auth/binding"
)

type TKEAuthConfigMaps struct {
	Informer v1.ConfigMapInformer
	Lister listersv1.ConfigMapLister
	Synced cache.InformerSynced
}

func NewTKEAuthConfigMaps(informer v1.ConfigMapInformer, lister listersv1.ConfigMapLister) *TKEAuthConfigMaps {
	authCfg := TKEAuthConfigMaps{
		Informer: informer,
		Lister:   lister,
		Synced:   informer.Informer().HasSynced,
	}

	return &authCfg
}

func ToTKEAuth(cfgMap *v12.ConfigMap) (*TKEAuth, error) {
	bindingName := cfgMap.Data[DataKeyBindingName]
	roleName := cfgMap.Data[DataKeyRoleName]

	usersStr := cfgMap.Data[DataKeyUsers]
	users := make([]string, 0)
	err := yaml.Unmarshal([]byte(usersStr), users)
	if err != nil {
		return nil, err
	}

	tkeAuth := &TKEAuth{
		BindingName: bindingName,
		RoleName:    roleName,
		Users:       users,
	}

	return tkeAuth, nil
}

// returns all deep-copied configMap with "tke-auth/binding" annotation attached
func (cfg *TKEAuthConfigMaps) GetTKEAuthConfigMaps() ([]*v12.ConfigMap, error) {
	cfg.waitUntilCacheSync()

	cfgMaps, err := cfg.Lister.List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	ret := make([]*v12.ConfigMap, 0)

	for _, cfgMap := range cfgMaps {
		if _, ok := cfgMap.Annotations[AnnotationKeyTKEAuthConfigMap]; ok {
			ret = append(ret, cfgMap.DeepCopy())
		}
	}

	return ret, nil
}

// wait until cache Synced
func (cfg *TKEAuthConfigMaps) waitUntilCacheSync() {
	stopCh := make(chan struct{})
	cache.WaitForCacheSync(stopCh, cfg.Synced)
	<-stopCh
}
