package internal

import (
	"encoding/json"
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

func NewClient(region string) (*tke.Client, error) {
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

func ConvertSubAccountIdToCommonNames(client *tke.Client, clusterId string, subAccountIds []string) ([]string, error) {
	// according to document, maximum subAccount per request is 50
	// check https://intl.cloud.tencent.com/document/product/457/41571?lang=en&pg= for more info.
	const maxSubAccountPerReq = 50

	CNs := make([]string, 0)

	for i := 0; i < len(subAccountIds)/maxSubAccountPerReq+1; i++ {
		length := min(maxSubAccountPerReq, len(subAccountIds)-i*maxSubAccountPerReq)

		req := tke.NewDescribeClusterCommonNamesRequest()
		req.ClusterId = &clusterId
		req.SubaccountUins = make([]*string, length)

		for j := 0; j < length; j++ {
			req.SubaccountUins[j] = &subAccountIds[i*maxSubAccountPerReq+j]
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
