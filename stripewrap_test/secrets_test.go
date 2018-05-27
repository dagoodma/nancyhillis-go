package stripewrap_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"testing"
)

func TestGetApiKey(t *testing.T) {
	publicKey, secretKey := stripewrap.GetApiKey()
	if len(publicKey) < 1 {
		t.Errorf("GetApiKey() == (empty string, _), want non-empty string for public key")
	}
	if len(secretKey) < 1 {
		t.Errorf("GetApiKey() == (_, empty string), want non-empty string for secret key")
	}
}
