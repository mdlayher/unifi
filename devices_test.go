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

	wantDevice := &AP{
		ID:      wantID,
		Adopted: wantAdopted,
		NICs:    []*NIC{},
		Radios:  []*Radio{},
		Stats: &APStats{
			All:    &WirelessStats{},
			User:   &WirelessStats{},
			Uplink: &WiredStats{},
		},
	}

	v := struct {
		// May be ap, switch, etc., but these unexported types
		// do not have a common interface.
		Devices []interface{} `json:"data"`
	}{
		Devices: []interface{}{
			ap{
				ID:      wantID,
				Adopted: wantAdopted,
				Type:    "uap",
			},
		},
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
		t.Fatalf("unexpected Device:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
