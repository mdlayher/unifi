package unifi

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestClientDevices(t *testing.T) {
	const (
		wantSite    = "default"
		wantID      = "abcdef123457890"
		wantAdopted = true
	)

	wantDevice := &Device{
		ID:      wantID,
		Adopted: wantAdopted,
	}

	v := struct {
		Devices []device `json:"data"`
	}{
		Devices: []device{{
			ID:      wantID,
			Adopted: wantAdopted,
		}},
	}

	c, done := testClient(t, testHandler(
		t,
		http.MethodGet,
		fmt.Sprintf("/api/s/%s/stat/device", wantSite),
		nil,
		v,
	))
	defer done()

	devices, err := c.Devices(wantSite)
	if err != nil {
		t.Fatalf("unexpected error from Client.Devices: %v", err)
	}

	if want, got := 1, len(devices); want != got {
		t.Fatalf("unexpected number of Devices:\n- want: %d\n-  got: %d",
			want, got)
	}

	if want, got := wantDevice, devices[0]; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Device:\n- want: %v\n-  got: %v",
			want, got)
	}
}
