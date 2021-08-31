package internal

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/informers/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	ConfigMapKeyBindings          = "bindings"
	AnnotationKeyTKEAuthConfigMap = "tke-auth/binding"
	ReSyncWaitTimeout             = time.Second * 10
)

type Binding struct {
	RoleName string   `yaml:"roleName"`
	Users    []string `yaml:"users"`
}

type TKEAuthConfig struct {
	configMap *v12.ConfigMap
	bindings  map[string]Binding
}

// GetBindings returns all Bindings from TKEAuthConfig. note: order is not deterministic
func (cfg *TKEAuthConfig) GetBindings() []Binding {
	arr := make([]Binding, 0)
	for _, binding := range cfg.bindings {
		arr = append(arr, binding)
	}

	return arr
}

type TKEAuthConfigMaps struct {
	informer v1.ConfigMapInformer
	lister   listersv1.ConfigMapLister
	synced   cache.InformerSynced
}

func GetBindingsFromConfigMap(configMap *v12.ConfigMap) (*TKEAuthConfig, error) {
	rawBindings, ok := configMap.Data[ConfigMapKeyBindings]
	if !ok {
		return nil, errors.Errorf("configMap %s doesn't have value %s\n", configMap.Name, ConfigMapKeyBindings)
	}

	bindings := &TKEAuthConfig{
		configMap: configMap.DeepCopy(),
		bindings:  make(map[string]Binding),
	}
	err := yaml.Unmarshal([]byte(rawBindings), &bindings.bindings)
	if err != nil {
		return nil, err
	}

	return bindings, nil
}

func UnMarshalYAMLToTKEAuthConfig(text string) (*TKEAuthConfig, error) {
	authCfg := TKEAuthConfig{}

	err := yaml.Unmarshal([]byte(text), &authCfg.bindings)
	if err != nil {
		return nil, err
	}

	return &authCfg, nil
}

func NewTKEAuthConfigMaps(informer v1.ConfigMapInformer, lister listersv1.ConfigMapLister) *TKEAuthConfigMaps {
	authCfg := TKEAuthConfigMaps{
		informer: informer,
		lister:   lister,
		synced:   informer.Informer().HasSynced,
	}

	return &authCfg
}

func (cfg *TKEAuthConfigMaps) GetAuthConfigMaps() ([]*v12.ConfigMap, error) {
	cfg.waitUntilCacheSync()

	cfgMaps, err := cfg.lister.List(labels.NewSelector())
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

// wait until cache synced
func (cfg *TKEAuthConfigMaps) waitUntilCacheSync() {
	stopCh := make(chan struct{})
	cache.WaitForCacheSync(stopCh, cfg.synced)
	<-stopCh
}
