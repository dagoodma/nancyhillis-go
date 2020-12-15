package studiojourney

import (
	//"fmt"
	//"strings"
	"testing"

	//"github.com/davecgh/go-spew/spew"

	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

func TestGetEnrollmentRowByEmail(t *testing.T) {
	cases := []struct {
		in        string
		wantRow   bool
		wantError bool
	}{
		{"davie.goodman.music@gmail.com", true, false},
		{"davie.goodman.music+tester@gmail.com", false, true},
		{"davie.goodman.music+tester6@gmail.com", false, true},
		{"davie.goodman.music", false, true},
		{"brelix@gmail.com", false, true},
		{"brelix+tester@gmail.com", false, true},
		{"brent.h.bailey@gmail.com", true, false},
		{"dagoodma@gmail.com", false, true},
		{"1", false, true},
		{"", false, true},
	}
	for _, c := range cases {
		r, err := studiojourney.GetEnrollmentRowByEmail(c.in)
		var gotRow = r != nil
		var gotError = err != nil
		if gotError && !c.wantError {
			t.Errorf("GetEnrollmentRowByEmail(%q) == (row=%t, error=%t), want (row=%t, error=%t), got err: %s",
				c.in, gotRow, gotError, c.wantRow, c.wantError, err)
		} else if !gotError && c.wantError ||
			!gotRow && c.wantRow || gotRow && !c.wantRow {
			t.Errorf("GetEnrollmentRowByEmail(%q) == (row=%t, error=%t), want (row=%t, error=%t)",
				c.in, gotRow, gotError, c.wantRow, c.wantError)
		} else if gotError && c.wantError {
			// Do nothing, but don't proceed into else
		} else {
			//fmt.Printf("%+v\n", r)
			//spew.Dump(r)
			// Ensure email matches
			if len(r) < studiojourney.EnrollmentSpreadsheetEmailCol {
				t.Errorf("GetEnrollmentRowByEmail(%q) expected at least %d rows, got only %d",
					c.in, c.in, len(r))
			}
			email := r[studiojourney.EnrollmentSpreadsheetEmailCol].Value
			// Ensure email matches input
			if c.in != email {
				t.Errorf("GetEnrollmentRowByEmail(%q) expected contact with email '%s', got: %s",
					c.in, c.in, email)
			}
		}
	}
}

func TestGetBillingRowByEmail(t *testing.T) {
	cases := []struct {
		in        string
		wantRow   bool
		wantError bool
	}{
		{"davie.goodman.music@gmail.com", true, false},
		{"davie.goodman.music+tester@gmail.com", false, true},
		{"davie.goodman.music+tester6@gmail.com", false, true},
		{"davie.goodman.music", false, true},
		{"brelix@gmail.com", false, true},
		{"brelix+tester@gmail.com", false, true},
		{"brent.h.bailey@gmail.com", true, false},
		{"dagoodma@gmail.com", false, true},
		{"nvkoch003@Gmail.com", true, false},
		{"1", false, true},
		{"", false, true},
	}
	for _, c := range cases {
		r, err := studiojourney.GetBillingRowByEmail(c.in)
		var gotRow = r != nil
		var gotError = err != nil
		if gotError && !c.wantError {
			t.Errorf("GetBillingRowByEmail(%q) == (row=%t, error=%t), want (row=%t, error=%t), got err: %s",
				c.in, gotRow, gotError, c.wantRow, c.wantError, err)
		} else if !gotError && c.wantError ||
			!gotRow && c.wantRow || gotRow && !c.wantRow {
			t.Errorf("GetBillingRowByEmail(%q) == (row=%t, error=%t), want (row=%t, error=%t)",
				c.in, gotRow, gotError, c.wantRow, c.wantError)
		} else if gotError && c.wantError {
			// Do nothing, but don't proceed into else
		} else {
			//fmt.Printf("%+v\n", r)
			//spew.Dump(r)
			// Ensure email matches
			if len(r) < studiojourney.EnrollmentSpreadsheetEmailCol {
				t.Errorf("GetBillingRowByEmail(%q) expected at least %d rows, got only %d",
					c.in, c.in, len(r))
			}
			email := r[studiojourney.BillingSpreadsheetEmailCol].Value
			// Ensure email matches input
			if c.in != email {
				t.Errorf("GetBillingRowByEmail(%q) expected contact with email '%s', got: %s",
					c.in, c.in, email)
			}
		}
	}
}
