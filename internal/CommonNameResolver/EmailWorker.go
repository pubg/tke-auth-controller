package CommonNameResolver

import (
	"example.com/tke-auth-controller/internal"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"github.com/thoas/go-funk"
	"k8s.io/klog/v2"
)

type Worker_Email struct {
	camClient *cam.Client
	tkeClient *tke.Client
	clusterId string

	userToRequestPerSecond int
}

func NewWorker_Email(camClient *cam.Client, tkeClient *tke.Client, clusterId string, userToRequestPerSecond int) *Worker_Email {
	return &Worker_Email{
		camClient:              camClient,
		tkeClient:              tkeClient,
		clusterId:              clusterId,
		userToRequestPerSecond: userToRequestPerSecond,
	}
}

func (worker *Worker_Email) ValueType() string {
	return "email"
}

func (worker *Worker_Email) ResolveCommonNames(users []*internal.User) error {
	for i := 0; i < len(users)/internal.SubAccountIdConversionUserCountPerRequest+1; i++ {
		start := i * internal.SubAccountIdConversionUserCountPerRequest
		end := funk.MinInt([]int{(i + 1) * internal.SubAccountIdConversionUserCountPerRequest, len(users)})
		length := end - start

		// fill names array for request
		names := make([]string, 0)
		for _, user := range users[start:end] {
			names = append(names, user.Value)
		}

		// convert names to subAccountIds for request
		subAccountIds, errs := internal.GetSubAccountIdOfUserIds(worker.camClient, worker.clusterId, names, worker.userToRequestPerSecond)
		if len(errs) > 0 {
			klog.Warningf("could not get subAccountId from email, ignoring. error: %s\n", errs)
		}

		// do actual request
		CNs, err := internal.ConvertSubAccountIdToCommonNames(worker.tkeClient, worker.clusterId, subAccountIds)
		if len(err) > 0 {
			klog.Warningf("could not get CommonNames from subAccountId, ignoring. error: %s\n", errs)
		}

		// convert user's subAccountId to CommonName
		for j := 0; j < length; j++ {
			(*users[start+j]).Value = CNs[j]
		}
	}

	return nil
}
