package activecampaign_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	//"fmt"
	"testing"
)

func TestGetContact(t *testing.T) {
	cases := []struct {
		in          string
		wantContact bool
		wantError   bool
	}{
		{"davie.goodman.music@gmail.com", false, true},
		//{"brelix@gmail.com", false, true},
		//{"dagoodma@gmail.com", true, false},
		//{"1", false, true},
		//{"", false, true},
	}
	for _, c := range cases {
		m, err := membermouse.GetMemberByEmail(c.in)
		if c.wantMember && m == nil || !c.wantMember && m != nil ||
			c.wantError && err == nil || !c.wantError && err != nil {
			var gotMember = m != nil
			var gotError = err != nil
			t.Errorf("GetMemberByEmail(%q) == (member=%t, error=%t), want (member=%t, error=%t)",
				c.in, gotMember, gotError, c.wantMember, c.wantError)
		}
	}
}
