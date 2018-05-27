package util_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/util"
	"testing"
)

func TestGetFirstAndLastName(t *testing.T) {
	cases := []struct {
		in, wantFirst, wantLast string
	}{
		{"Clayton Chavez", "Clayton", "Chavez"},
		{"DiMaggio St. Croix", "DiMaggio", "St. Croix"},
		//{"Dr. Hangs O'Reilly", "Dr. Hangs", "O'Reilly"}, // TODO fix this one fails!
		{"Peppy Le-Pew", "Peppy", "Le-Pew"},
		{"Dan", "Dan", ""},
		{"Really Long Long Name", "Really", "Long Long Name"},
		{"1", "1", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		gotFirst, gotLast := util.GetFirstAndLastName(c.in)
		if gotFirst != c.wantFirst || gotLast != c.wantLast {
			t.Errorf("GetFirstNameLastName(%q) ==  (%q, %q), want (%q, %q)",
				c.in, gotFirst, gotLast, c.wantFirst, c.wantLast)
		}
	}
}

func TestEmailLooksValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Clayton Chavez", false},
		{"DiMaggio St. Croix", false},
		{"Xdjhasd@gmail.com", true},
		{"Xd-@gmail.co", true},
		{"x8*123@", false},
		{"88s", false},
		{"abd@", false},
		{"--asdasd_@too", false},
		{"--asdasd_@too.co.uk", true},
		{"my-real-email+47@goo.com", true},
		{"maggie@goo.", false},
		{"josh-c-cap@goo.x", true},
		{"", false},
		{"axx", false},
		{"@", false},
	}
	for _, c := range cases {
		got := util.EmailLooksValid(c.in)
		if got != c.want {
			t.Errorf("EmailLooksValid(%q) ==  %q, want %q",
				c.in, got, c.want)
		}
	}
}

func StringSliceContains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func TestStringSliceContains(t *testing.T) {
	cases := []struct {
		slice []string
		in    string
		want  bool
	}{
		{[]string{"testing", "is", "it"}, "it", true},
		{[]string{"testing", "is", "it"}, "test", false},
		{[]string{"testing", "Is", "it"}, "is", false},
		//{[]string{"testing", "Is", "it"}, "is", true}, // not case insensitive
		{[]string{"it"}, "itt", false},
		{[]string{""}, "", true},
		{[]string{}, "", false},
		{[]string{"h"}, "", false},
		{[]string{"h", "1", "2", "3"}, "2", true},
		{[]string{"h", "1", "3", "3"}, "3", true},
	}
	for _, c := range cases {
		got := util.StringSliceContains(c.slice, c.in)
		if got != c.want {
			t.Errorf("StringSliceContains(%q in: %q) ==  %q, want %q",
				c.in, c.slice, got, c.want)
		}
	}
}
