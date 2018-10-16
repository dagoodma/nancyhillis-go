package activecampaign_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	//"fmt"
	"testing"
)

func TestGetContactsByEmail(t *testing.T) {
	cases := []struct {
		in           string
		wantContacts bool
		wantError    bool
	}{
		{"davie.goodman.music@gmail.com", false, false},
		{"davie.goodman.music+tester@gmail.com", true, false},
		{"davie.goodman.music", false, true},
		{"brelix@gmail.com", false, false},
		{"brelix+tester@gmail.com", true, false},
		{"dagoodma@gmail.com", false, false},
		{"1", false, true},
		{"", false, true},
	}
	for _, c := range cases {
		r, err := activecampaign.GetContactsByEmail(c.in)
		var gotContactList = r != nil
		var gotError = err != nil
		var gotContacts = r != nil && r.Metadata.Total > 0
		if gotError && !c.wantError || !gotError && c.wantError || !gotContactList && !gotError {
			t.Errorf("GetContactsByEmail(%q) == (contactList=%t, error=%t), want (contactList=%t, error=%t)",
				c.in, gotContactList, gotError, c.wantContacts, c.wantError)
		} else if gotContacts && !c.wantContacts {
			t.Errorf("GetContactsByEmail(%q) == (contacts count=%d, error=%t), expected no contacts, but got some",
				c.in, r.Metadata.Total, gotError)
		} else if !gotContactList && c.wantContacts {
			t.Errorf("GetContactsByEmail(%q) == (contacts=%t, error=%t), expected contacts but got none",
				c.in, gotContacts, gotError)
		} else if gotError && c.wantError {
			// Do nothing, but don't proceed into else
		} else {
			if r.Metadata.Total < 1 && c.wantContacts {
				t.Errorf("GetContactsByEmail(%q) returned no contacts, expected at least 1 contact",
					c.in, r.Metadata.Total)
			}
		}
	}
}
