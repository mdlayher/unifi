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
	"time"
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
		Stats: &DeviceStats{
			All:    &WirelessStats{},
			User:   &WirelessStats{},
			Uplink: &WiredStats{},
		},
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
	"name": "AP",
	"ethernet_table": [
		{
			"mac": "de:ad:be:ef:de:ad",
			"name": "eth0"
		}
	],
	"na-num_sta": 3,
	"na-user-num_sta": 2,
	"na-guest-num_sta": 1,
	"ng-num_sta": 3,
	"ng-user-num_sta": 2,
	"ng-guest-num_sta": 1,
	"radio_table": [
		{
			"builtin_ant_gain": 1,
			"builtin_antenna": true,
			"max_txpower": 10,
			"min_txpower": 1,
			"name": "wlan0",
			"radio": "ng"
		},
		{
			"builtin_ant_gain": 1,
			"builtin_antenna": true,
			"max_txpower": 10,
			"min_txpower": 1,
			"name": "wlan1",
			"radio": "na"
		}
	],
	"serial": "deadbeef0123456789",
	"site_id": "default",
	"stat": {
		"bytes": 100,
		"rx_bytes": 80,
		"rx_packets": 4,
		"tx_bytes": 20,
		"tx_dropped": 1,
		"tx_packets": 1,
		"user-rx_bytes": 80,
		"user-rx_packets": 4,
		"user-tx_bytes": 20,
		"user-tx_dropped": 1,
		"user-tx_packets": 1
	},
	"uplink": {  
		"full_duplex": true,
		"ip": "0.0.0.0",
		"mac": "de:ad:be:ef:00:00",
		"max_speed": 1000,
		"name": "eth0",
		"netmask": "0.0.0.0",
		"num_port": 2,
		"rx_bytes": 81,
		"rx_dropped": 11023,
		"rx_errors": 0,
		"rx_multicast": 0,
		"rx_packets": 5,
		"speed": 1000,
		"tx_bytes": 21,
		"tx_dropped": 0,
		"tx_errors": 0,
		"tx_packets": 2,
		"type": "wire",
		"up": true
	},
	"uptime": 61,
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
				Name:  "AP",
				NICs: []*NIC{{
					MAC:  net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad},
					Name: "eth0",
				}},
				Radios: []*Radio{
					{
						BuiltInAntenna:     true,
						BuiltInAntennaGain: 1,
						MaxTXPower:         10,
						MinTXPower:         1,
						Name:               "wlan0",
						Radio:              radio24GHz,
						Stats: &RadioStationsStats{
							NumberStations:      3,
							NumberUserStations:  2,
							NumberGuestStations: 1,
						},
					},
					{
						BuiltInAntenna:     true,
						BuiltInAntennaGain: 1,
						MaxTXPower:         10,
						MinTXPower:         1,
						Name:               "wlan1",
						Radio:              radio5GHz,
						Stats: &RadioStationsStats{
							NumberStations:      3,
							NumberUserStations:  2,
							NumberGuestStations: 1,
						},
					},
				},
				Serial: "deadbeef0123456789",
				SiteID: "default",
				Stats: &DeviceStats{
					TotalBytes: 100,
					All: &WirelessStats{
						ReceiveBytes:    80,
						ReceivePackets:  4,
						TransmitBytes:   20,
						TransmitDropped: 1,
						TransmitPackets: 1,
					},
					User: &WirelessStats{
						ReceiveBytes:    80,
						ReceivePackets:  4,
						TransmitBytes:   20,
						TransmitDropped: 1,
						TransmitPackets: 1,
					},
					Uplink: &WiredStats{
						ReceiveBytes:    81,
						ReceivePackets:  5,
						TransmitBytes:   21,
						TransmitPackets: 2,
					},
				},
				Uptime:  61 * time.Second,
				Version: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			d := new(Device)
			err := d.UnmarshalJSON(tt.b)
			if want, got := errStr(tt.err), errStr(err); !strings.Contains(got, want) {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if tt.err != nil {
				return
			}
			if err != nil {
				t.Fatalf("Error parsing json: %v", err)
			}

			if want, got := tt.d, d; !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected Device:\n- want: %+v\n-  got: %+v",
					want, got)
			}
		})
	}
}
