package unifi

import (
	"net/http"
	"reflect"
	"testing"
)

func TestClientSites(t *testing.T) {
	wantSite := &Site{
		Name:        "default",
		Description: "Company",
	}

	v := struct {
		Sites []*Site `json:"data"`
	}{
		Sites: []*Site{wantSite},
	}

	c, done := testClient(t, testHandler(t, http.MethodGet, "/api/self/sites", nil, v))
	defer done()

	sites, err := c.Sites()
	if err != nil {
		t.Fatalf("unexpected error from Client.Sites: %v", err)
	}

	if want, got := 1, len(sites); want != got {
		t.Fatalf("unexpected number of Sites:\n- want: %d\n-  got: %d",
			want, got)
	}

	if want, got := wantSite, sites[0]; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Site:\n- want: %v\n-  got: %v",
			want, got)
	}
}
