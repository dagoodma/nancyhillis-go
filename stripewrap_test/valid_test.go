package stripewrap_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"testing"
)

func TestCustomerIdLooksValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"cus_XXXXXXasda", true},
		{"cus_asjkdhaskj", true},
		{"xxx", false},
		{"1", false},
		{"", false},
	}
	for _, c := range cases {
		got := stripewrap.CustomerIdLooksValid(c.in)
		if got != c.want {
			t.Errorf("CustomerIdLooksValid(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTokenIdLooksValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"tok_XXXXXXasda", true},
		{"tok_asjkdhaskj", true},
		{"xxx", false},
		{"1", false},
		{"", false},
	}
	for _, c := range cases {
		got := stripewrap.TokenIdLooksValid(c.in)
		if got != c.want {
			t.Errorf("TokenIdLooksValid(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
