package gossip

import "testing"

func TestIsMatch(t *testing.T) {

	for _, test := range []struct {
		Matcher     string
		Key         string
		ShouldMatch bool
	}{
		{
			Matcher:     "*",
			Key:         "Something",
			ShouldMatch: true,
		},
		{
			Matcher:     "customer.*",
			Key:         "driver.update",
			ShouldMatch: false,
		},
		{
			Matcher:     "customer.*",
			Key:         "customer.update",
			ShouldMatch: true,
		},
		{
			Matcher:     "*.update",
			Key:         "driver.update",
			ShouldMatch: true,
		},
		{
			Matcher:     "*.update",
			Key:         "driver.create",
			ShouldMatch: false,
		},
	} {
		req := &RequestHandler{
			Key: test.Matcher,
		}
		res := req.IsMatch(test.Key)
		if res != test.ShouldMatch {
			t.Errorf("Expected match to be %t, got %t", test.ShouldMatch, res)
		}
	}
}
