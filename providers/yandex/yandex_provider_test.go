// SPDX-License-Identifier: Apache-2.0

package yandex

import (
	"strings"
	"testing"
)

func TestYandexProviderInitClearsStateOnMissingFolderID(t *testing.T) {
	t.Setenv("YC_TOKEN", "env-token")
	t.Setenv("YC_SERVICE_ACCOUNT_KEY_FILE", "env-key")
	t.Setenv("YC_FOLDER_ID", "")
	provider := YandexProvider{
		token:              "old-token",
		saKeyFileOrContent: "old-key",
		folderID:           "old-folder",
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing folder ID error")
	}
	if !strings.Contains(err.Error(), "set YC_FOLDER_ID env var") {
		t.Fatalf("expected folder ID error, got %q", err)
	}
	if provider.token != "" {
		t.Fatalf("token = %q, want empty", provider.token)
	}
	if provider.saKeyFileOrContent != "" {
		t.Fatalf("saKeyFileOrContent = %q, want empty", provider.saKeyFileOrContent)
	}
	if provider.folderID != "" {
		t.Fatalf("folderID = %q, want empty", provider.folderID)
	}
}

func TestYandexProviderInitUsesEnvFolderIDWhenArgEmpty(t *testing.T) {
	t.Setenv("YC_TOKEN", "env-token")
	t.Setenv("YC_SERVICE_ACCOUNT_KEY_FILE", "env-key")
	t.Setenv("YC_FOLDER_ID", "env-folder")
	var provider YandexProvider

	if err := provider.Init([]string{""}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.folderID != "env-folder" {
		t.Fatalf("folderID = %q, want env-folder", provider.folderID)
	}
	if provider.token != "env-token" {
		t.Fatalf("token = %q, want env-token", provider.token)
	}
	if provider.saKeyFileOrContent != "env-key" {
		t.Fatalf("saKeyFileOrContent = %q, want env-key", provider.saKeyFileOrContent)
	}
}
