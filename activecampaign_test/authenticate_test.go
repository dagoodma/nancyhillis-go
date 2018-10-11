package activecampaign_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	"testing"
)

func TestAuthenticateWithCredentials(t *testing.T) {
	apiUrl, apiToken := activecampaign.GetApiCredentials()

	err := activecampaign.AuthenticateWithCredentials(apiUrl, apiToken)

	if err != nil {
		t.Errorf("AuthenticateWithCredentials() == (error), expected no error but got: %s",
			err)
	}
}
