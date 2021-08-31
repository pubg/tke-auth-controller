package internal

import (
	"context"
	"github.com/thoas/go-funk"
	v14 "k8s.io/api/rbac/v1"
	v15 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/informers/rbac/v1"
	v13 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	v12 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"log"
)

type TKEAuthClusterRoleBindings struct {
	Informer v1.ClusterRoleBindingInformer
	Lister v12.ClusterRoleBindingLister
	Synced cache.InformerSynced

	crbIface v13.ClusterRoleBindingInterface
}

const (
	AnnotationKeyManagedTKEAuthCRB = "tke-auth/managed-by"
	AnnotationValueManagedTKEAuthCRB = "tke-auth"
)

func NewTKEAuthClusterRoleBinding(informer v1.ClusterRoleBindingInformer, lister v12.ClusterRoleBindingLister) *TKEAuthClusterRoleBindings {
	crb := &TKEAuthClusterRoleBindings{
		Informer: informer,
		Lister:   lister,
		Synced:   informer.Informer().HasSynced,
	}

	return crb
}

func (TKEAuthCRB *TKEAuthClusterRoleBindings) UpsertClusterRoleBindings(newCRBs []*v14.ClusterRoleBinding) error {
	TKEAuthCRB.waitUntilCacheSync()

	oldCRBs, err := TKEAuthCRB.getClusterRoleBindings()
	if err != nil {
		return err
	}

	deletions, _ := funk.Difference(oldCRBs, newCRBs)
	additions, _ := funk.Difference(newCRBs, oldCRBs)
	updates := funk.Join(newCRBs, oldCRBs, funk.InnerJoin)

	err = TKEAuthCRB.deleteCRBs(deletions.([]*v14.ClusterRoleBinding))
	if err != nil {
		return err
	}
	err = TKEAuthCRB.addCRBs(additions.([]*v14.ClusterRoleBinding))
	if err != nil {
		return err
	}
	err = TKEAuthCRB.updateCRBs(updates.([]*v14.ClusterRoleBinding))
	if err != nil {
		return err
	}

	return nil
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
		checkClusterRoleBindingIsManaged(crb)
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
	stopCh := make(chan struct{})
	cache.WaitForCacheSync(stopCh, TKEAuthCRB.Synced)
	<-stopCh
}
