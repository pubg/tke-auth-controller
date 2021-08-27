package main

import (
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
)

func init() {
	flag.StringVar(&masterURL, "masterURL", "", "masterURL of kubernetes cluster.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path of kubeconfig.")
	flag.Parse()

	if masterURL == "" || kubeconfig == "" {
		flag.PrintDefaults()
	}
}

func main() {
	log.Printf("current os: %s\n", runtime.GOOS)

	// setup for graceful shutdown
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("cannot create kubeconfig, %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
 	if err != nil {
 		log.Fatalf("cannot create kubeClient: %s", err.Error())
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second * 30)

	controller, err := NewController(kubeClient, kubeInformerFactory.Core().V1().ConfigMaps(), kubeInformerFactory.Rbac().V1().ClusterRoleBindings())
	if err != nil {
		log.Fatalf("cannot create controller: %s", err.Error())
	}

	kubeInformerFactory.Start(stopCh)

	if err = controller.Run(stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}

	return
}
