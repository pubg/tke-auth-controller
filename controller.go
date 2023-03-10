package main

import (
	"example.com/tke-auth-controller/internal"
	"example.com/tke-auth-controller/internal/CommonNameResolver"
	log "example.com/tke-auth-controller/log"
	"fmt"
	"github.com/pkg/errors"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

/*
- https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md
- v1.ConfigMapInformer 를 이용하여 변경 이벤트 수신
- Informer 에 event Handler 를 달아서 설정 가능
- utilruntime 을 이용해서 crash 를 방지 (defer utilruntime.HandleCrash())
- custom resource lister 등을 만들어서 사용도 가능하다? https://github.com/kubernetes/sample-controller/blob/master/pkg/generated/listers/samplecontroller/v1alpha1/foo.go
- github.com/kubernetes/code-generator 라는게 있다.
	- CustomResourceDefinition 을 사용할 때, native, versioned client, informersv1, other helpers 를 작성하는데 도움을 줌
	- User-provider API Server 등을 만들 때 도움을 줌
*/

const (
	reSyncWaitTimeout = time.Millisecond * 500
)

type Controller struct {
	kubeClient                 kubernetes.Interface
	tkeAuthConfigMap           *internal.TKEAuthConfigMaps
	tkeAuthClusterRoleBindings *internal.TKEAuthClusterRoleBindings

	syncAllClusterRoleBindingTimer *time.Timer

	clusterId string
	tkeClient *tke.Client

	commonNameResolver *CommonNameResolver.CommonNameResolver
}

func NewController(kubeClient kubernetes.Interface, tkeAuthCfg *internal.TKEAuthConfigMaps, tkeAuthCRB *internal.TKEAuthClusterRoleBindings, tkeClient *tke.Client, clusterId string, CNResolver *CommonNameResolver.CommonNameResolver) (*Controller, error) {
	ctl := &Controller{
		kubeClient:                     kubeClient,
		tkeAuthConfigMap:               tkeAuthCfg,
		tkeAuthClusterRoleBindings:     tkeAuthCRB,
		syncAllClusterRoleBindingTimer: nil,
		tkeClient:                      tkeClient,
		clusterId:                      clusterId,
		commonNameResolver:             CNResolver,
	}

	ctl.tkeAuthConfigMap.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.onConfigMapAdded,
		UpdateFunc: ctl.onConfigMapUpdated,
		DeleteFunc: ctl.onConfigMapDeleted,
	})

	return ctl, nil
}

func (ctl *Controller) onConfigMapAdded(new interface{}) {
	configMap, ok := new.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("failed trying to cast new object to configMap, new: %s\n", new)
		return
	}

	if !v12.HasAnnotation(configMap.ObjectMeta, internal.AnnotationKeyTKEAuthConfigMap) {
		return
	}

	ctl.reserveReSyncTimer()
	klog.V(log.VerboseLevel).Infof("received configMap added event, name: %s\n", configMap.Name)
}

func (ctl *Controller) onConfigMapUpdated(old, new interface{}) {
	oldConfigMap, ok := old.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("failed trying to cast old object to oldConfigMap, new: %s\n", new)
		return
	}

	newConfigMap, ok := new.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("failed trying to cast new object to oldConfigMap, new: %s\n", new)
		return
	}

	oldCfgMapIsManaged := v12.HasAnnotation(oldConfigMap.ObjectMeta, internal.AnnotationKeyTKEAuthConfigMap)
	newCfgMapIsManaged := v12.HasAnnotation(newConfigMap.ObjectMeta, internal.AnnotationKeyTKEAuthConfigMap)

	if !oldCfgMapIsManaged && !newCfgMapIsManaged {
		return
	} else if oldCfgMapIsManaged && !newCfgMapIsManaged {
		klog.Warningf("configMap %s has annotation \"managed-by\" before, but is deleted.\n", newConfigMap.Name)
	}

	ctl.reserveReSyncTimer()
	klog.V(log.VerboseLevel).Infof("received configMap changed event, name: %s\n", newConfigMap.Name)
}

func (ctl *Controller) onConfigMapDeleted(old interface{}) {
	configMap, ok := old.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("failed trying to cast old object to configMap, old: %s\n", old)
		return
	}

	if !v12.HasAnnotation(configMap.ObjectMeta, internal.AnnotationKeyTKEAuthConfigMap) {
		return
	}

	ctl.reserveReSyncTimer()
	klog.V(log.VerboseLevel).Infof("received configMap deleted event, name: %s\n", configMap.Name)
}

func (ctl *Controller) reserveReSyncTimer() {
	timer := &ctl.syncAllClusterRoleBindingTimer
	if ctl.syncAllClusterRoleBindingTimer != nil {
		(*timer).Reset(reSyncWaitTimeout)
	} else {
		*timer = time.AfterFunc(reSyncWaitTimeout, ctl.syncAllClusterRoleBinding)
	}
}

func (ctl *Controller) syncAllClusterRoleBinding() {
	// 1. get all TKE-Auth config maps
	cfgMaps, err := ctl.tkeAuthConfigMap.GetTKEAuthConfigMaps()
	if err != nil {
		klog.Error(errors.Wrap(err, "Cannot get AuthConfigMaps from cluster"))
	}
	klog.V(log.VerboseLevel).Infof("got %d configMaps.\n", len(cfgMaps))

	// 2. convert to tkeAuth
	tkeAuths := make([]*internal.TKEAuth, 0)
	for _, cfg := range cfgMaps {
		tkeAuth, err := internal.ToTKEAuth(cfg)
		if err != nil {
			klog.Error(err)
			return
		} else {
			tkeAuths = append(tkeAuths, tkeAuth)
		}
	}

	// 3. convert subAccountId to CommonNames
	for _, tkeAuth := range tkeAuths {
		err := (ctl.commonNameResolver).ResolveCommonNames(tkeAuth.Users)
		if err != nil {
			klog.Error(err)
			return
		}
	}

	// 4. convert to ClusterRoleBinding
	TKEAuthCRBs := make([]*v13.ClusterRoleBinding, 0)
	for _, tkeAuth := range tkeAuths {
		crb := tkeAuth.ToClusterRoleBinding()
		TKEAuthCRBs = append(TKEAuthCRBs, crb)
	}

	// 5. upsert CRBs
	err = ctl.tkeAuthClusterRoleBindings.UpsertClusterRoleBindings(TKEAuthCRBs)
	if err != nil {
		klog.Error(err)
	} else {
		klog.Infoln("ClusterRoleBindings updated.")
	}
}

func (ctl *Controller) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	klog.Infoln("Starting Controller.")

	klog.V(4).Infoln("Waiting for informer caches to sync.")
	if ok := cache.WaitForCacheSync(stopCh, ctl.tkeAuthConfigMap.Synced, ctl.tkeAuthClusterRoleBindings.Synced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync.\n")
	}

	klog.Infoln("Controller running...")
	<-stopCh
	klog.Infoln("Controller stopped.")

	return nil
}
