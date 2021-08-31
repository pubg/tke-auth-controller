package main

import (
	"example.com/tke-auth-controller/internal"
	"example.com/tke-auth-controller/internal/signals"
	"flag"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"runtime"
	"time"
)

var (
	masterURL string
	kubeconfig string
	regionName string
	clusterId string
)

func init() {
	flag.StringVar(&masterURL, "masterURL", "", "masterURL of kubernetes cluster.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path of kubeconfig.")
	flag.StringVar(&regionName, "regionName", "", "region Name. eg: ap-seoul")
	flag.StringVar(&clusterId, "clusterId", "", "cluster Id of target.")
	flag.Parse()

	if masterURL == "" || kubeconfig == "" || regionName == "" || clusterId == "" {
		flag.PrintDefaults()
	}
}

func main() {
	log.Printf("current os: %s\n", runtime.GOOS)

	// setup for graceful shutdown
	stopCh := signals.SetupSignalHandler()

	tkeClient, err := internal.NewClient(regionName)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("cannot create kubeconfig, %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
 	if err != nil {
 		log.Fatalf("cannot create kubeClient: %s", err.Error())
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second * 30)
	tkeAuthCfg := internal.NewTKEAuthConfigMaps(informerFactory.Core().V1().ConfigMaps(), informerFactory.Core().V1().ConfigMaps().Lister())
	tkeAuthCRB := internal.NewTKEAuthClusterRoleBinding(informerFactory.Rbac().V1().ClusterRoleBindings(), informerFactory.Rbac().V1().ClusterRoleBindings().Lister())

	controller, err := NewController(kubeClient, tkeAuthCfg, tkeAuthCRB, tkeClient, clusterId)
	if err != nil {
		log.Fatalf("cannot create controller: %s", err.Error())
	}

	informerFactory.Start(stopCh)

	if err = controller.Run(stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}

	return
}
