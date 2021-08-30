package internal

import (
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/informers/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

type TKEAuthConfigMap struct {
	informer v1.ConfigMapInformer
	lister listersv1.ConfigMapLister
	synced cache.InformerSynced
}

const (
	AnnotationKeyTKEAuthConfigMap = "tke-auth/binding"
	ReSyncWaitTimeout = time.Second * 10
)

func NewTKEAuthConfigMap(informer v1.ConfigMapInformer, lister listersv1.ConfigMapLister) *TKEAuthConfigMap {
	authCfg := TKEAuthConfigMap{
		informer: informer,
		lister:   lister,
		synced:   informer.Informer().HasSynced,
	}

	return &authCfg
}

func (cfg *TKEAuthConfigMap) GetAuthConfigMaps() ([]*v12.ConfigMap, error) {
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
func (cfg *TKEAuthConfigMap) waitUntilCacheSync() {
	stopCh := make(chan struct{})
	cache.WaitForCacheSync(stopCh, cfg.synced)
	<- stopCh
}