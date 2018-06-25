package stripewrap_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	//"fmt"
	"strings"
	"testing"
)

func TestGetCanceledSubList(t *testing.T) {
	cases := []struct {
		in        string
		wantCount int
	}{
		{"cus_CPAPKAesBIjQ4O", 3},
		{"cus_CP91ALH2IZINXM", 1},
		{"cus_CyIeNt9YVxQHaT", 1},
		{"cus_Ct7u0bdzPdx8a8", 0},
		{"cus_asjkdhaskj", 0},
		{"xxx", 0},
		{"1", 0},
	}
	for _, c := range cases {
		got := stripewrap.GetCanceledSubList(c.in)
		// Count the number of charges
		var cnt = 0
		for got.Next() {
			c := got.Subscription()
			_ = c
			cnt = cnt + 1
			//fmt.Printf("%d: %s (%v)\n", cnt, c, c)
		}
		if cnt != c.wantCount {
			t.Errorf("len(GetCanceledSubList(%q) == %d, want %d", c.in, cnt, c.wantCount)
		}
	}
}

func TestGetLastCanceledSubWithPrefix(t *testing.T) {
	cases := []struct {
		in        string
		inPrefix  string
		wantError bool
	}{
		{"cus_CPAPKAesBIjQ4O", "sj-", false},
		{"cus_CPAPKAesBIjQ4O", "taj-", true},
		{"cus_CP91ALH2IZINXM", "sj-", false},
		{"cus_CyIeNt9YVxQHaT", "sj-", false},
		{"cus_Ct7u0bdzPdx8a8", "sj-", true},
		{"cus_asjkdhaskj", "sj-", true},
		{"xxx", "x", true},
		{"1", "poo", true},
	}
	for _, c := range cases {
		got, gotErr := stripewrap.GetLastCanceledSubWithPrefix(c.in, c.inPrefix)
		//fmt.Printf("got: %v\n", got)
		if gotErr != nil && !c.wantError {
			t.Errorf("GetLastCanceledSubWithPrefix(%q, %q) == (sub=%q, error=%v), want (sub=*, error=%t)",
				c.in, c.inPrefix, got, gotErr, c.wantError)
		}
		if !c.wantError {
			if got == nil {
				t.Errorf("GetLastCanceledSubWithPrefix(%q, %q) == (sub=nil, error=nil), want (sub=*, error=%t)",
					c.in, c.inPrefix, c.wantError)
			}
			if got.Status != "canceled" || got.EndedAt <= 0 || got.CanceledAt <= 0 {
				t.Errorf("GetLastCanceledSubWithPrefix(%q, %q) returned non-canceled sub=%q",
					c.in, c.inPrefix, got)
			}
			if !strings.HasPrefix(got.Plan.ID, c.inPrefix) {
				t.Errorf("GetLastCanceledSubWithPrefix(%q, %q) returned canceled plan sub=%q with prefix=%q, want prefix=%q",
					c.in, c.inPrefix, got, got.Plan.ID, c.inPrefix)
			}
		}
	}
}
