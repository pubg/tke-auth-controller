package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informersv1 "k8s.io/client-go/informers/core/v1"
	rbacv1 "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
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

type Controller struct {
	kubeClient        kubernetes.Interface
	configMapInformer informersv1.ConfigMapInformer
	configMapLister   listersv1.ConfigMapLister
	configMapSynced   cache.InformerSynced
	clusterRoleBindingLister rbacv1.ClusterRoleBindingInformer
	clusterRoleBindingSynced cache.InformerSynced
}

func NewController(kubeClient kubernetes.Interface, configMapInformer informersv1.ConfigMapInformer, clusterRoleBindingInformer rbacv1.ClusterRoleBindingInformer) (*Controller, error) {
	ctl := &Controller{
		kubeClient: kubeClient,
		configMapInformer: configMapInformer,
		configMapLister: configMapInformer.Lister(),
		configMapSynced: configMapInformer.Informer().HasSynced,
		clusterRoleBindingLister: clusterRoleBindingInformer,
		clusterRoleBindingSynced: clusterRoleBindingInformer.Informer().HasSynced,
	}

	ctl.configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctl.onConfigMapAdded,
	})

	return ctl, nil
}

func (ctl *Controller) process() {

}

func (ctl *Controller) onConfigMapAdded(new interface{}) {
	log.Printf("onConfigMapAdded: %s\n", new)
}

func (ctl *Controller) onConfigMapUpdated(old, new interface{}) {
	log.Printf("onConfigMapChanged old: %s\n", old)
	log.Printf("onConfigMapChanged new: %s\n", new)
}

func (ctl *Controller) onConfigMapDeleted(old interface{}) {
	log.Printf("onConfigMapDeleted %s\n", old)
}

func (ctl *Controller) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	log.Println("Starting Controller.")

	log.Println("Waiting for informer caches to sync.")
	if ok := cache.WaitForCacheSync(stopCh, ctl.configMapSynced, ctl.clusterRoleBindingSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync.\n")
	}

	go wait.Until(ctl.RunWorker, time.Second, stopCh)

	log.Println("Started workers.")
	<- stopCh
	log.Println("Shutdown workers.")

	return nil
}

func (ctl *Controller) RunWorker() {
	// do nothing?
}