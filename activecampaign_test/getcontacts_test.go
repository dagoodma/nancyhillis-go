package activecampaign_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	//"fmt"
	"testing"
)

func TestGetContactsByEmail(t *testing.T) {
	cases := []struct {
		in          string
		wantContact bool
		wantError   bool
	}{
		{"davie.goodman.music@gmail.com", false, true},
		{"davie.goodman.music+tester@gmail.com", true, false},
		{"brelix@gmail.com", false, true},
		{"brelix+tester@gmail.com", true, false},
		{"dagoodma@gmail.com", false, true},
		{"1", false, true},
		{"", false, true},
	}
	for _, c := range cases {
		r, err := activecampaign.GetContactsByEmail(c.in)
		if c.wantContact && r == nil || !c.wantContact && r != nil ||
			c.wantError && err == nil || !c.wantError && err != nil {
			var gotContact = r != nil
			var gotError = err != nil
			t.Errorf("GetContactsByEmail(%q) == (contact=%t, error=%t), want (contact=%t, error=%t)",
				c.in, gotContact, gotError, c.wantContact, c.wantError)
		}
	}
}
