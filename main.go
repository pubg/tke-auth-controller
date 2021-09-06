package main

import (
	"example.com/tke-auth-controller/internal"
	"example.com/tke-auth-controller/internal/CommonNameResolver"
	"example.com/tke-auth-controller/internal/signals"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/regions"
	v20180525 "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	masterURL                   string
	kubeconfig                  string
	regionName                  string
	clusterName                 string
	clusterId                   string
	reSyncInterval              int
	userToEmailRequestPerSecond int
	tkeClient                   *v20180525.Client
	camClient                   *cam.Client
)

func init() {
	flag.StringVar(&masterURL, "masterURL", "", "masterURL of kubernetes cluster.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path of kubeconfig.")
	flag.StringVar(&regionName, "regionName", "", "region Name. eg: ap-seoul")
	flag.StringVar(&clusterName, "clusterName", "", "name of cluster.")
	flag.StringVar(&clusterId, "clusterId", "", "cluster Id of target.")
	flag.IntVar(&reSyncInterval, "reSyncInterval", 60*5, "interval (second) to reSync event trigger. does not effect reSync on configMap changes.")
	flag.IntVar(&userToEmailRequestPerSecond, "userToEmailReqPerSec", 5, "user to email request per second. high value might exceed API Call limit.")
	flag.Parse()

	if clusterName == "" && clusterId == "" {
		log.Println("both clusterName and clusterId is empty, you should provide at least one value.")
		flag.PrintDefaults()
		log.Printf("received arguments: %s\n", os.Args)
		os.Exit(1)
	}

	if regionName == "" {
		log.Println("required attribute: regionName is empty.")
		flag.PrintDefaults()
		log.Printf("received arguments: %s\n", os.Args)
		os.Exit(1)
	}

	setupTKEClient()
}

func setupTKEClient() {
	log.Printf("current region: %s, cluster name: %s\n", regionName, clusterName)

	if !isValidRegionName(regionName) {
		log.Fatalf("region: %s is not a valid regionName.\n", regionName)
	}

	if regionName == "" {
		log.Fatalln("regionName is empty. you should provide valid region name.")
	}

	var err error
	tkeClient, err = internal.NewTKEClient(regionName)
	if err != nil {
		log.Fatalln(err)
	}

	camClient, err = internal.NewCAMClient(regionName)
	if err != nil {
		log.Fatalln(err)
	}

	if clusterId == "" || masterURL == "" {
		log.Printf("clusterId: %s, masterURL: %s, value is empty. fetching via TKE API.\n", clusterId, masterURL)

		req := v20180525.NewDescribeClustersRequest()
		res, err := tkeClient.DescribeClusters(req)
		if err != nil {
			log.Fatalln(err)
		}

		for _, cluster := range res.Response.Clusters {
			if *cluster.ClusterName == clusterName { // found
				clusterId = *cluster.ClusterId
				masterURL = fmt.Sprintf("https://%s.ccs.tencent-cloud.com", clusterId)
				break
			}
		}
	} else {
		log.Printf("using values from command arguments. clusterId: %s, masterURL: %s\n", clusterId, masterURL)
	}

	if clusterId == "" || masterURL == "" {
		log.Fatalf("Cannnot get clusterId or masterURL of given clusterName: \"%s\" in region: \"%s\"\n", clusterName, regionName)
	}
}

func main() {
	log.Printf("current os: %s\n", runtime.GOOS)

	// setup for graceful shutdown
	stopCh := signals.SetupSignalHandler()

	cfg, err := getClusterConfig()
	if err != nil {
		log.Fatalf("cannot create kubeconfig, %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("cannot create kubeClient: %s", err.Error())
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*10)
	tkeAuthCfg := internal.NewTKEAuthConfigMaps(informerFactory.Core().V1().ConfigMaps(), informerFactory.Core().V1().ConfigMaps().Lister())
	tkeAuthCRB := internal.NewTKEAuthClusterRoleBinding(informerFactory.Rbac().V1().ClusterRoleBindings(), informerFactory.Rbac().V1().ClusterRoleBindings().Lister(), kubeClient.RbacV1().ClusterRoleBindings(), stopCh)
	commonNameResolver := CommonNameResolver.NewCommonNameResolver()

	subAccountIdResolveWorker := CommonNameResolver.NewWorker_SubAccountId(tkeClient, clusterId)
	commonNameResolver.AddWorker(subAccountIdResolveWorker)
	emailResolveWorker := CommonNameResolver.NewWorker_Email(camClient, tkeClient, clusterId, userToEmailRequestPerSecond)
	commonNameResolver.AddWorker(emailResolveWorker)

	controller, err := NewController(kubeClient, tkeAuthCfg, tkeAuthCRB, tkeClient, clusterId, commonNameResolver, reSyncInterval)
	if err != nil {
		log.Fatalf("cannot create controller: %s", err.Error())
	}

	informerFactory.Start(stopCh)

	if err = controller.Run(stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}

	return
}

func getClusterConfig() (*rest.Config, error) {
	cfg, err := getOutClusterConfig()
	if err == nil {
		return cfg, nil
	}

	cfg, err = getInClusterConfig()
	if err == nil {
		return cfg, nil
	}

	return nil, errors.Errorf("cannot get config from kubeconfig or serviceAccount. err: %s\n", err)
}

func getInClusterConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	return cfg, err
}

func getOutClusterConfig() (*rest.Config, error) {
	var kubeConfigPath string
	if kubeconfig != "" {
		kubeConfigPath = kubeconfig
	} else {
		home, _ := os.UserHomeDir()
		kubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfigPath)

	return cfg, err
}

func isValidRegionName(regionName string) bool {
	switch regionName {
	case regions.Bangkok:
		return true
	case regions.Beijing:
		return true
	case regions.Chengdu:
		return true
	case regions.Chongqing:
		return true
	case regions.Guangzhou:
		return true
	case regions.GuangzhouOpen:
		return true
	case regions.HongKong:
		return true
	case regions.Mumbai:
		return true
	case regions.Seoul:
		return true
	case regions.Shanghai:
		return true
	case regions.Nanjing:
		return true
	case regions.ShanghaiFSI:
		return true
	case regions.ShenzhenFSI:
		return true
	case regions.Singapore:
		return true
	case regions.Tokyo:
		return true
	case regions.Frankfurt:
		return true
	case regions.Moscow:
		return true
	case regions.Ashburn:
		return true
	case regions.SiliconValley:
		return true
	case regions.Toronto:
		return true
	}

	return false
}
