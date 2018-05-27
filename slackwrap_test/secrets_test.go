package slackwrap_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
	"testing"
)

func TestGetApiToken(t *testing.T) {
	apiToken := slackwrap.GetApiToken()
	if len(apiToken) < 1 {
		t.Errorf("GetApiToken() == blank , want non-empty string")
	}
}

func TestGetCommandTokens(t *testing.T) {
	commandTokens := slackwrap.GetCommandTokens()
	if len(commandTokens) < 1 {
		t.Errorf("GetCommandTokens() is empty, want non-empty map")
	}
	for k, v := range commandTokens {
		if len(k) < 1 {
			t.Errorf("GetCommandTokens() gave an empty key")
		}
		if len(v) < 1 {
			t.Errorf("GetCommandTokens() gave an empty value for key: %s", k)
		}
	}
}
