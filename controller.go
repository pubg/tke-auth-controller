package main

import (
	"example.com/tke-auth-controller/internal"
	"fmt"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	informersv1 "k8s.io/client-go/informers/core/v1"
	rbacv1 "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"log"
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
	AnnotationKeyTKEAuthConfigMap = "tke-auth/binding-user-data"
	resyncWaitTimeout = time.Second * 1
)

type Controller struct {
	kubeClient        kubernetes.Interface
	configMapInformer informersv1.ConfigMapInformer
	configMapLister   listersv1.ConfigMapLister
	configMapSynced   cache.InformerSynced
	clusterRoleBindingLister rbacv1.ClusterRoleBindingInformer
	clusterRoleBindingSynced       cache.InformerSynced
	syncAllClusterRoleBindingTimer *time.Timer

	TKEClients *internal.TKEClients
}

func NewController(kubeClient kubernetes.Interface, configMapInformer informersv1.ConfigMapInformer, clusterRoleBindingInformer rbacv1.ClusterRoleBindingInformer) (*Controller, error) {
	ctl := &Controller{
		kubeClient: kubeClient,
		configMapInformer: configMapInformer,
		configMapLister: configMapInformer.Lister(),
		configMapSynced: configMapInformer.Informer().HasSynced,
		clusterRoleBindingLister: clusterRoleBindingInformer,
		clusterRoleBindingSynced: clusterRoleBindingInformer.Informer().HasSynced,
		TKEClients: internal.NewTKEClients(),
	}

	ctl.configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctl.onConfigMapAdded,
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

	if v12.HasAnnotation(configMap.ObjectMeta, AnnotationKeyTKEAuthConfigMap) {
		klog.Infof("configMap %s has annotation %s.\n", configMap.Name, AnnotationKeyTKEAuthConfigMap)
		ctl.reserveReSyncTimer()
	}
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

	if v12.HasAnnotation(oldConfigMap.ObjectMeta, AnnotationKeyTKEAuthConfigMap) {
		klog.Infof("oldConfigMap %s has annotation %s.\n", oldConfigMap.Name, AnnotationKeyTKEAuthConfigMap)
		ctl.reserveReSyncTimer()
	}

	if v12.HasAnnotation(newConfigMap.ObjectMeta, AnnotationKeyTKEAuthConfigMap) {
		klog.Infof("oldConfigMap %s has annotation %s.\n", oldConfigMap.Name, AnnotationKeyTKEAuthConfigMap)
		ctl.reserveReSyncTimer()
	}
}

func (ctl *Controller) onConfigMapDeleted(old interface{}) {
	configMap, ok := old.(*v1.ConfigMap)
	if !ok {
		klog.Errorf("failed trying to cast old object to configMap, old: %s\n", old)
		return
	}

	if v12.HasAnnotation(configMap.ObjectMeta, AnnotationKeyTKEAuthConfigMap) {
		klog.Infof("configMap %s has annotation %s.\n", configMap.Name, AnnotationKeyTKEAuthConfigMap)
		ctl.reserveReSyncTimer()
	}
}

func (ctl *Controller) reserveReSyncTimer() {
	timer := &ctl.syncAllClusterRoleBindingTimer
	if ctl.syncAllClusterRoleBindingTimer != nil {
		(*timer).Reset(resyncWaitTimeout)
	} else {
		*timer = time.AfterFunc(resyncWaitTimeout, ctl.syncAllClusterRoleBinding)
	}
}

func (ctl *Controller) syncAllClusterRoleBinding() {
	configs, err := ctl.configMapLister.List(labels.NewSelector())
	if err != nil {
		klog.Error(errors.Wrap(err, "Cannot list configMaps"))
		ctl.reserveReSyncTimer()
		return
	}


	annotatedConfigs := funk.Filter(configs, func(cfgMap *v1.ConfigMap) bool {
		return v12.HasAnnotation(cfgMap.ObjectMeta, AnnotationKeyTKEAuthConfigMap)
	}) // []v12.ConfigMap

	_ = annotatedConfigs

	ctl.kubeClient.RbacV1().ClusterRoleBindings().Apply()

	//ctl.kubeClient.RbacV1().ClusterRoleBindings().Apply()

	return
}

func (ctl *Controller) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	log.Println("Starting Controller.")

	log.Println("Waiting for informer caches to sync.")
	if ok := cache.WaitForCacheSync(stopCh, ctl.configMapSynced, ctl.clusterRoleBindingSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync.\n")
	}

	log.Println("Controller running...")
	<- stopCh
	log.Println("Controller stopped.")

	return nil
}
