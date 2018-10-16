package activecampaign_test

import (
	"fmt"
	"strings"
	"testing"

	//"github.com/davecgh/go-spew/spew"

	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
)

/*
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
*/
/*
func TestGetContact(t *testing.T) {
*/

func TestGetContactByEmail(t *testing.T) {
	cases := []struct {
		in          string
		wantContact bool
		wantError   bool
	}{
		{"davie.goodman.music@gmail.com", false, true},
		{"davie.goodman.music+tester@gmail.com", true, false},
		{"davie.goodman.music", false, true},
		{"brelix@gmail.com", false, true},
		{"brelix+tester@gmail.com", true, false},
		{"dagoodma@gmail.com", false, true},
		{"1", false, true},
		{"", false, true},
	}
	for _, c := range cases {
		r, err := activecampaign.GetContactByEmail(c.in)
		var gotContact = r != nil
		var gotError = err != nil
		if gotError && !c.wantError {
			t.Errorf("GetContactByEmail(%q) == (contact=%t, error=%t), want (contact=%t, error=%t), got err: %s",
				c.in, gotContact, gotError, c.wantContact, c.wantError, err)
		} else if !gotError && c.wantError ||
			!gotContact && c.wantContact || gotContact && !c.wantContact {
			t.Errorf("GetContactByEmail(%q) == (contact=%t, error=%t), want (contact=%t, error=%t)",
				c.in, gotContact, gotError, c.wantContact, c.wantError)
		} else if gotError && c.wantError {
			// Do nothing, but don't proceed into else
		} else {
			//fmt.Printf("%+v\n", r)
			//spew.Dump(r)
			// Ensure they have an id
			if len(r.Id) < 1 {
				t.Errorf("GetContactByEmail(%q) expected contact with non-empty ID, got: '%s'",
					c.in, r.Id)
			}
			// Ensure email matches input
			if c.in != r.Email {
				t.Errorf("GetContactByEmail(%q) expected contact with email '%s', got: %s",
					c.in, c.in, r.Email)
			}
		}
	}
}

func TestGetContactProfileUrlById(t *testing.T) {
	cases := []struct {
		in          string
		expectedOut string
	}{
		{"123", "https://nancyhillis.activehosted.com/app/contacts/123"},
		{"xx", "https://nancyhillis.activehosted.com/app/contacts/xx"},
		{"9999999", "https://nancyhillis.activehosted.com/app/contacts/9999999"},
		{"-1", "https://nancyhillis.activehosted.com/app/contacts/-1"},
		{"0", "https://nancyhillis.activehosted.com/app/contacts/0"},
		{"", "https://nancyhillis.activehosted.com/app/contacts/"},
	}
	for _, c := range cases {
		out := activecampaign.GetContactProfileUrlById(c.in)
		if len(out) < 1 {
			t.Errorf("GetContactProfileUrlById(%q) == (%s), expected a non-empty string",
				c.in, out)
		} else {
			if !strings.HasPrefix(out, "https") {
				t.Errorf("GetContactProfileUrlById(%q) == (%s), expected a URL starting with 'https'",
					c.in, out)
			}

			if out != c.expectedOut {
				t.Errorf("GetContactProfileUrlById(%q) == (%s), expected: %s",
					c.in, out, c.expectedOut)
			}
		}
	}
}

func TestGetContactProfileUrlByEmail(t *testing.T) {
	cases := []struct {
		in        string
		wantError bool
	}{
		{"davie.goodman.music@gmail.com", true},
		{"davie.goodman.music+tester@gmail.com", false},
		{"davie.goodman.music", true},
		{"brelix@gmail.com", true},
		{"brelix+tester@gmail.com", false},
		{"dagoodma@gmail.com", true},
		{"1", true},
		{"", true},
	}
	for _, c := range cases {
		url, err := activecampaign.GetContactProfileUrlByEmail(c.in)
		fmt.Printf("Here with in=%s url=%s\n", c.in, url)
		var gotError = err != nil
		if gotError && !c.wantError {
			t.Errorf("GetContactProfileUrlByEmail(%q) == (url=%s, error=%t), wanted (url=..., error=%t), got err: %s",
				c.in, url, gotError, c.wantError, err)
		} else if !gotError && c.wantError {
			t.Errorf("GetContactProfileUrlByEmail(%q) == (url=%s, error=%t), wanted (url=..., error=%t)",
				c.in, url, gotError, c.wantError)
		} else if !c.wantError {
			if len(url) < 1 {
				t.Errorf("GetContactProfileUrlByEmail(%q) == (\"%s\"), expected a non-empty string",
					c.in, url)
			} else {
				if !strings.HasPrefix(url, "https") {
					t.Errorf("GetContactProfileUrlById(%q) == (\"%s\"), expected a URL starting with 'https'",
						c.in, url)
				}
			}
		}
	}
}
