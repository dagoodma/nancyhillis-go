package membermouse_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"testing"
)

func TestGetApiCredentials(t *testing.T) {
	apiKey, apiPassword := membermouse.GetApiCredentials()
	if len(apiKey) < 1 {
		t.Errorf("GetApiCredentials() == (empty string, _), want non-empty string for api key")
	}
	if len(apiPassword) < 1 {
		t.Errorf("GetApiCredentials() == (_, empty string), want non-empty string for api password")
	}
}
