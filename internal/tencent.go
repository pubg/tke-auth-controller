package internal

import (
	"encoding/json"
	"github.com/pkg/errors"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
	"os"
	"path"
	"strconv"
	"time"
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

func NewTKEClient(region string) (*tke.Client, error) {
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

func NewCAMClient(region string) (*cam.Client, error) {
	credProviders := common.NewProviderChain([]common.Provider{common.DefaultEnvProvider(), common.DefaultProfileProvider(), common.DefaultCvmRoleProvider(), TencentIntlProfileProvider{}})
	cred, err := credProviders.GetCredential()
	if err != nil {
		return nil, err
	}

	client, err := cam.NewClient(cred, region, profile.NewClientProfile())
	if err != nil {
		return nil, err
	}

	return client, nil
}

const (
	SubAccountIdConversionUserCountPerRequest = 100
)

// ConvertSubAccountIdToCommonNames accepts subAccountId array, returns same length of commonName array
// the value of index is original subAccountId if somehow the request is failed.
func ConvertSubAccountIdToCommonNames(client *tke.Client, clusterId string, subAccountIds []string, apiCallPerSecond int) ([]string, []error) {
	CNs := make([]string, 0)
	errs := make([]error, 0)

	for _, id := range subAccountIds {
		req := tke.NewDescribeClusterCommonNamesRequest()
		req.ClusterId = &clusterId
		req.SubaccountUins = []*string{
			&id,
		}

		res, err := client.DescribeClusterCommonNames(req)
		if err != nil || res.Response == nil || len(res.Response.CommonNames) == 0 {
			errs = append(errs, err, errors.Wrapf(err, "could not get commonName, subAccountId: %s\n", id))
			CNs = append(CNs, id)
		} else {
			CNs = append(CNs, *res.Response.CommonNames[0].CN)
		}

		time.Sleep(time.Second / time.Duration(apiCallPerSecond))
	}

	return CNs, errs
}

func GetSubAccountIdOfUserName(client *cam.Client, clusterId string, userId string) (*string, error) {
	req := cam.NewGetUserRequest()
	req.Name = &userId

	res, err := client.GetUser(req)
	if err != nil {
		return nil, err
	}

	str := strconv.FormatUint(*res.Response.Uin, 10)

	return &str, nil
}

// GetSubAccountIdOfUserIds accepts userId array, returns subAccountId array
// if request fails, the value of index will be replaced to empty string.
func GetSubAccountIdOfUserIds(client *cam.Client, clusterId string, userIds []string, requestPerSecond int) ([]string, []error) {
	users := make([]string, 0)
	errs := make([]error, 0)

	for _, name := range userIds {
		userId, err := GetSubAccountIdOfUserName(client, clusterId, name)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "could not get user info, userId: %s\n", name))
			users = append(users, name) // give original name if request failed. (empty string is not allowed, k8s will throw error)
		} else {
			users = append(users, *userId)
		}

		time.Sleep(time.Second / time.Duration(requestPerSecond))
	}

	return users, errs
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
