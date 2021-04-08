package activecampaign_test

import (
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
    "testing"
)

// TODO support fetching this and remove hard coded id
var ChangedEmailCustomFieldId = "71"
var TestSecretsPath = "ac_secrets.yml"

func TestUpdateContactCustomField(t *testing.T) {
    ac.SecretsFilePath = TestSecretsPath

	cases := []struct {
		emailIn         string
		customFieldIn   string
		customValueIn   string
        wantError       bool
	}{
		{"brelix+25@gmail.com", ChangedEmailCustomFieldId, "testingggg221", false},
	}
	for _, c := range cases {
        // Propagate changes (email) through to system
        // - Active Campaign
        c1, err := ac.GetContactByEmail(c.emailIn)
        gotError := err != nil
        gotContact := c1 != nil
        if gotError && !c.wantError {
			t.Errorf("GetContactByEmail(%q) == (contact=%t, error=%t), want (contact=%t, error=%t), got err: %s",
				c.emailIn, gotContact, gotError, true, c.wantError, err)
            return
        }

        err = ac.UpdateContactCustomField(c1, c.customFieldIn, c.customValueIn)
        gotError = err != nil
        if gotError && !c.wantError {
			t.Errorf("UpdateContactCustomField(%q [id=%s], %q, %q) == (error=%t), want (error=%t), got err: %s",
				c.emailIn, c1.Id, c.customFieldIn, c.customValueIn, gotError, c.wantError, err)
            return
        }
    }
}
