package CommonNameResolver

import (
	"example.com/tke-auth-controller/internal"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"github.com/thoas/go-funk"
)

type Worker_SubAccountId struct {
	client    *tke.Client
	clusterId string
}

const (
	userCountPerRequest = 100
)

func NewWorker_SubAccountId(client *tke.Client, clusterId string) *Worker_SubAccountId {
	return &Worker_SubAccountId{
		client:    client,
		clusterId: clusterId,
	}
}

func (worker *Worker_SubAccountId) ValueType() string {
	return "subAccountId"
}

func (worker *Worker_SubAccountId) ResolveCommonNames(users []*internal.User) error {
	for i := 0; i < len(users)/userCountPerRequest+1; i++ {
		start := i * userCountPerRequest
		end := funk.MinInt([]int{(i + 1) * userCountPerRequest, len(users)})
		length := end - start

		// fill accountIds array for request
		subAccountIds := make([]string, 0)
		for _, user := range users[start:end] {
			subAccountIds = append(subAccountIds, user.Value)
		}

		// do actual request
		CNs, err := internal.ConvertSubAccountIdToCommonNames(worker.client, worker.clusterId, subAccountIds)
		if err != nil {
			return err
		}

		// convert user's subAccountId to CommonName
		for j := 0; j < length; j++ {
			(*users[start+j]).Value = CNs[j]
		}
	}

	return nil
}
