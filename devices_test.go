package unifi

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestClientDevices(t *testing.T) {
	const (
		wantSite    = "default"
		wantID      = "abcdef123457890"
		wantAdopted = true
	)
	var (
		wantInformIP = net.IPv4(192, 168, 1, 1)
	)

	wantDevice := &Device{
		ID:       wantID,
		Adopted:  wantAdopted,
		InformIP: wantInformIP,
		NICs:     []*NIC{},
		Radios:   []*Radio{},
	}

	v := struct {
		Devices []device `json:"data"`
	}{
		Devices: []device{{
			ID:       wantID,
			Adopted:  wantAdopted,
			InformIP: wantInformIP.String(),
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

	// For easy comparison
	wantDevice.InformURL = nil
	devices[0].InformURL = nil

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

func TestDeviceUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		desc string
		b    []byte
		d    *Device
		err  error
	}{
		{
			desc: "invalid JSON",
			b:    []byte(`<>`),
			err:  errors.New("invalid character"),
		},
		{
			desc: "invalid inform IP",
			b:    []byte(`{"inform_ip":"foo"}`),
			err:  errors.New("failed to parse inform IP"),
		},
		{
			desc: "invalid NIC MAC",
			b:    []byte(`{"inform_ip":"192.168.1.1","ethernet_table":[{"mac":"foo"}]}`),
			err:  errors.New("invalid MAC address"),
		},
		{
			desc: "OK",
			b: bytes.TrimSpace([]byte(`
{
	"_id": "abcdef1234567890",
	"adopted": true,
	"inform_ip": "192.168.1.1",
	"inform_url": "http://192.168.1.1:8080/inform",
	"model": "uap1000",
	"ethernet_table": [
		{
			"mac": "de:ad:be:ef:de:ad",
			"name": "eth0"
		}
	],
	"radio_table": [
		{
			"builtin_ant_gain": 1,
			"builtin_antenna": true,
			"max_txpower": 10,
			"min_txpower": 1,
			"name": "wlan0",
			"radio": "ng"
		}
	],
	"serial": "deadbeef0123456789",
	"site_id": "default",
	"version": "1.0.0"
}
`)),
			d: &Device{
				ID:       "abcdef1234567890",
				Adopted:  true,
				InformIP: net.IPv4(192, 168, 1, 1),
				InformURL: func() *url.URL {
					u, err := url.Parse("http://192.168.1.1:8080/inform")
					if err != nil {
						t.Fatal("failed to parse inform URL")
					}

					return u
				}(),
				Model: "uap1000",
				NICs: []*NIC{{
					MAC:  net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad},
					Name: "eth0",
				}},
				Radios: []*Radio{{
					BuiltInAntenna:     true,
					BuiltInAntennaGain: 1,
					MaxTXPower:         10,
					MinTXPower:         1,
					Name:               "wlan0",
					Radio:              "ng",
				}},
				Serial:  "deadbeef0123456789",
				SiteID:  "default",
				Version: "1.0.0",
			},
		},
	}

	for i, tt := range tests {
		t.Logf("[%02d] test %q", i, tt.desc)

		d := new(Device)
		err := d.UnmarshalJSON(tt.b)
		if want, got := errStr(tt.err), errStr(err); !strings.Contains(got, want) {
			t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
				want, got)
		}
		if err != nil {
			continue
		}

		if want, got := tt.d, d; !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected Device:\n- want: %+v\n-  got: %+v",
				want, got)
		}
	}
}
