package activecampaign_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	"strings"
	"testing"
)

var ApiUrlPrefix = "https://"

func TestGetApiCredentials(t *testing.T) {
	apiUrl, apiToken := activecampaign.GetApiCredentials()
	if len(apiUrl) < 1 {
		t.Errorf("GetApiCredentials() == (empty string, _), want non-empty string for api url")
	}
	if !strings.HasPrefix(apiUrl, ApiUrlPrefix) {
		t.Errorf("GetApiCredentials() == (%s, _), want string starting with '%s' for api url",
			apiUrl, ApiUrlPrefix)
	}
	if len(apiToken) < 1 {
		t.Errorf("GetApiCredentials() == (_, empty string), want non-empty string for api token")
	}
}
