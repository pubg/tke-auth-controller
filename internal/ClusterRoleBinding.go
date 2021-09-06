package internal

import (
	"context"
	"fmt"
	v14 "k8s.io/api/rbac/v1"
	v15 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/informers/rbac/v1"
	v13 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	v12 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"log"
)

type TKEAuthClusterRoleBindings struct {
	Informer v1.ClusterRoleBindingInformer
	Lister   v12.ClusterRoleBindingLister
	Synced   cache.InformerSynced

	crbIface v13.ClusterRoleBindingInterface

	stopCh <-chan struct{}
}

const (
	AnnotationKeyManagedTKEAuthCRB   = "tke-auth/managed-by"
	AnnotationValueManagedTKEAuthCRB = "tke-auth"
)

func NewTKEAuthClusterRoleBinding(informer v1.ClusterRoleBindingInformer, lister v12.ClusterRoleBindingLister, crbIface v13.ClusterRoleBindingInterface, stopCh <-chan struct{}) *TKEAuthClusterRoleBindings {
	crb := &TKEAuthClusterRoleBindings{
		Informer: informer,
		Lister:   lister,
		Synced:   informer.Informer().HasSynced,
		crbIface: crbIface,
		stopCh:   stopCh,
	}

	return crb
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) UpsertClusterRoleBindings(newCRBs []*v14.ClusterRoleBinding) error {
	TKEAuthCRB.waitUntilCacheSync()

	oldCRBs, err := TKEAuthCRB.getClusterRoleBindings()
	if err != nil {
		return err
	}

	deletions := difference(oldCRBs, newCRBs)
	additions := difference(newCRBs, oldCRBs)
	updates := getUpdates(newCRBs, oldCRBs)
	klog.Infof("crb changed. add: %d, update: %d, delete: %d\n", len(additions), len(updates), len(deletions))

	err = TKEAuthCRB.deleteCRBs(deletions)
	if err != nil {
		return err
	}
	err = TKEAuthCRB.addCRBs(additions)
	if err != nil {
		return err
	}
	err = TKEAuthCRB.updateCRBs(updates)
	if err != nil {
		return err
	}

	return nil
}

// difference returns A - B in set, key is Name
func difference(a, b []*v14.ClusterRoleBinding) []*v14.ClusterRoleBinding {
	aSet := make(map[string]*v14.ClusterRoleBinding)
	bSet := make(map[string]*v14.ClusterRoleBinding)

	for _, crb := range a {
		aSet[crb.Name] = crb
	}

	for _, crb := range b {
		bSet[crb.Name] = crb
	}

	for key, _ := range bSet {
		if _, ok := aSet[key]; ok {
			delete(aSet, key)
		}
	}

	arr := arrayToMap(aSet)
	return arr
}

func getUpdates(new, old []*v14.ClusterRoleBinding) []*v14.ClusterRoleBinding {
	oldSet := map[string]*v14.ClusterRoleBinding{}

	// create oldSet
	for _, crb := range old {
		oldSet[crb.Name] = crb
	}

	updates := make([]*v14.ClusterRoleBinding, 0)

	for _, newCrb := range new {
		oldCrb, ok := oldSet[newCrb.Name]

		if ok {
			oldCrbCopy := oldCrb.DeepCopy()
			newCrbCopy := newCrb.DeepCopy()
			oldCrbCopy.Subjects = newCrbCopy.Subjects
			oldCrbCopy.RoleRef = newCrbCopy.RoleRef
			updates = append(updates, oldCrbCopy)
		}
	}

	return updates
}
// intersection returns AnB in set, key is Name, uses b's value for array
func intersection(a, b []*v14.ClusterRoleBinding) []*v14.ClusterRoleBinding {
	aSet := make(map[string]*v14.ClusterRoleBinding)
	bSet := make(map[string]*v14.ClusterRoleBinding)

	for _, crb := range a {
		aSet[crb.Name] = crb
	}

	for _, crb := range b {
		bSet[crb.Name] = crb
	}

	for key, _ := range bSet {
		if _, ok := aSet[key]; !ok {
			delete(bSet, key)
		}
	}

	arr := arrayToMap(bSet)
	return arr
}

func arrayToMap(m map[string]*v14.ClusterRoleBinding) []*v14.ClusterRoleBinding {
	arr := make([]*v14.ClusterRoleBinding, 0)

	for _, val := range m {
		arr = append(arr, val)
	}

	return arr
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) addCRBs(CRBs []*v14.ClusterRoleBinding) error {
	crbIface := TKEAuthCRB.crbIface
	for _, crb := range CRBs {
 		crb.Annotations[AnnotationKeyManagedTKEAuthCRB] = AnnotationValueManagedTKEAuthCRB
		_, err := crbIface.Create(context.TODO(), crb, v15.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) updateCRBs(CRBs []*v14.ClusterRoleBinding) error {
	crbIface := TKEAuthCRB.crbIface
	for _, crb := range CRBs {
		_, err := crbIface.Update(context.TODO(), crb, v15.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) deleteCRBs(CRBs []*v14.ClusterRoleBinding) error {
	crbIface := TKEAuthCRB.crbIface
	for _, crb := range CRBs {
		checkClusterRoleBindingIsManaged(crb)
		err := crbIface.Delete(context.TODO(), crb.Name, v15.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// check clusterRoleBinding has managed annotation, throws panic if not.
func checkClusterRoleBindingIsManaged(crb *v14.ClusterRoleBinding) {
	if _, ok := crb.Annotations[AnnotationKeyManagedTKEAuthCRB]; !ok {
		log.Panicf("tried to modify ClusterRoleBinding name: %s but it's not managed by TKE-Auth controller.\n", crb.Name)
	}
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) getClusterRoleBindings() ([]*v14.ClusterRoleBinding, error) {
	TKEAuthCRB.waitUntilCacheSync()

	CRBs, err := TKEAuthCRB.Lister.List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	ret := make([]*v14.ClusterRoleBinding, 0)

	for _, crb := range CRBs {
		if _, ok := crb.Annotations[AnnotationKeyManagedTKEAuthCRB]; ok {
			ret = append(ret, crb.DeepCopy())
		}
	}

	return ret, nil
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) waitUntilCacheSync() {
	retryCount := 0
	for {
		klog.Infoln(fmt.Sprintf("Waiting TKEAuthClusterRoleBindings cache to be synced... retryCount: %d", retryCount))
		if cache.WaitForCacheSync(TKEAuthCRB.stopCh, TKEAuthCRB.Synced) {
			klog.Infoln("TKEAuthClusterRoleBindings cache synced.")
			break
		} else {
			retryCount += 1

			if retryCount > syncRetryCountLimit {
				panic("Cannot sync ClusterRoleBinding.")
			}
		}
	}
}
