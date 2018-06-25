package stripewrap_test

import (
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	//"fmt"
	//"reflect"
	"testing"
)

func TestGetChargeList(t *testing.T) {
	cases := []struct {
		in        string
		wantCount int
	}{
		{"cus_CPAPKAesBIjQ4O", 1},
		{"cus_CP91ALH2IZINXM", 0},
		{"cus_asjkdhaskj", 0},
		{"xxx", 0},
		{"1", 0},
	}
	for _, c := range cases {
		got := stripewrap.GetChargeList(c.in)
		//fmt.Printf("Here with: %v\n", got)
		//val := reflect.Indirect(reflect.ValueOf(got.Iter))
		//fmt.Println(val.Type().Field(0).Name)
		//gotCount := len(i.values)
		// Count the number of charges
		var cnt = 0
		for got.Next() {
			c := got.Charge()
			_ = c
			cnt = cnt + 1
			//fmt.Printf("%d: %s (%v)\n", cnt, c, c)
		}
		/*
			if cnt < 1 || c == nil {
				msg := fmt.Sprintf("No charges starting with: %s", descriptionPrefix)
				return nil, errors.New(msg)
			}
		*/
		if cnt != c.wantCount {
			t.Errorf("len(GetChargeList(%q)) == %d, want %d", c.in, cnt, c.wantCount)
		}
	}
}
