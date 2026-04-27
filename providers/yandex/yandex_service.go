// SPDX-License-Identifier: Apache-2.0

package yandex

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mitchellh/go-homedir"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

type YandexService struct { //nolint
	terraformutils.Service
}

func (y *YandexService) InitSDK() (*ycsdk.SDK, error) {
	if saKeyOrContent := y.Args[KeySaKeyFileOrContent].(string); saKeyOrContent != "" {
		contents, _, err := pathOrContents(saKeyOrContent)
		if err != nil {
			return nil, fmt.Errorf("error loading credentials: %w", err)
		}

		key, err := iamKeyFromJSONContent(contents)
		if err != nil {
			return nil, err
		}
		serviceAccountKey, err := ycsdk.ServiceAccountKey(key)
		if err != nil {
			return nil, err
		}
		return ycsdk.Build(context.Background(), ycsdk.Config{
			Credentials: serviceAccountKey},
		)
	}

	if cToken := y.Args[KeyToken].(string); cToken != "" {
		if strings.HasPrefix(cToken, "t1.") && strings.Count(cToken, ".") == 2 {
			return ycsdk.Build(context.Background(), ycsdk.Config{
				Credentials: ycsdk.NewIAMTokenCredentials(cToken)},
			)
		}
		return ycsdk.Build(context.Background(), ycsdk.Config{
			Credentials: ycsdk.OAuthToken(cToken),
		})
	}

	if sa := ycsdk.InstanceServiceAccount(); checkServiceAccountAvailable(context.Background(), sa) {
		return ycsdk.Build(context.Background(), ycsdk.Config{
			Credentials: sa,
		})
	}

	return nil, fmt.Errorf("one of 'YC_TOKEN' or 'YC_SERVICE_ACCOUNT_KEY_FILE' env variable should be specified; if you are inside compute instance, you can attach service account to it in order to authenticate via instance service account")
}

func pathOrContents(poc string) (string, bool, error) {
	if len(poc) == 0 {
		return poc, false, nil
	}

	path := poc
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return path, true, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		return string(contents), true, err
	}

	return poc, false, nil
}

func iamKeyFromJSONContent(content string) (*iamkey.Key, error) {
	key := &iamkey.Key{}
	err := json.Unmarshal([]byte(content), key)
	if err != nil {
		return nil, fmt.Errorf("service account JSON key unmarshal fail: %w", err)
	}
	return key, nil
}

func checkServiceAccountAvailable(ctx context.Context, sa ycsdk.NonExchangeableCredentials) bool {
	dialer := net.Dialer{Timeout: 50 * time.Millisecond}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ycsdk.InstanceMetadataAddr, "80"))
	if err != nil {
		return false
	}
	_ = conn.Close()
	_, err = sa.IAMToken(ctx)
	return err == nil
}
