package internal

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"os"
	"path"
)

type TencentIntlProfileProvider struct{}

func (t TencentIntlProfileProvider) GetCredential() (common.CredentialIface, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	buf, err := os.ReadFile(path.Join(home, "/.tccli/default.credential"))
	if err != nil {
		return nil, err
	}

	rawCred := map[string]interface{}{}
	err = json.Unmarshal(buf, &rawCred)
	if err != nil {
		return nil, err
	}

	return &common.Credential{
		SecretId:  rawCred["secretId"].(string),
		SecretKey: rawCred["secretKey"].(string),
	}, nil
}

type TKEClients struct {
	clients map[string]*tke.Client
}

func NewTKEClients() *TKEClients {
	return &TKEClients{
		clients: make(map[string]*tke.Client),
	}
}

func (t *TKEClients) GetClientOfRegion(region string) (client *tke.Client, err error) {
	if t.clients[region] == nil {
		client, err := newClient(region)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot create client of region %s\n", region)
		}

		t.clients[region] = client
	}

	return t.clients[region], nil
}

func newClient(region string) (*tke.Client, error) {
	credProviders := common.NewProviderChain([]common.Provider{common.DefaultEnvProvider(), common.DefaultProfileProvider(), common.DefaultCvmRoleProvider(), TencentIntlProfileProvider{}})
	cred, err := credProviders.GetCredential()
	if err != nil {
		return nil, err
	}

	client, err := tke.NewClient(cred, region, profile.NewClientProfile())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (t *TKEClients) ConvertSubAccountIdToCommonNames(region string, clusterId string, subAccountIds []string) ([]string, error) {
	client, err := t.GetClientOfRegion(region)
	if err != nil {
		return nil, err
	}

	// according to document, maximum subAccount per request is 50
	// check https://intl.cloud.tencent.com/document/product/457/41571?lang=en&pg= for more info.
	const maxSubAccountPerReq = 50

	CNs := make([]string, len(subAccountIds))

	for i := 0; i < len(subAccountIds) / maxSubAccountPerReq; i++ {
		length := min(maxSubAccountPerReq, len(subAccountIds) - i * maxSubAccountPerReq)

		req := tke.NewDescribeClusterCommonNamesRequest()
		req.ClusterId = &clusterId
		req.SubaccountUins = make([]*string, length)

		for j := 0; j < length; j++ {
			req.SubaccountUins[j] = &subAccountIds[i * maxSubAccountPerReq + j]
		}

		res, err := client.DescribeClusterCommonNames(req)
		if err != nil {
			return nil, err
		}

		for _, commonName := range res.Response.CommonNames {
			CNs = append(CNs, *commonName.CN)
		}
	}

	return CNs, nil
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
