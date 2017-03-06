package unifi

import (
	"bytes"
	"errors"
	"net"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAPUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		desc string
		b    []byte
		ap   *AP
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
		"user-tx_packets": 1,
		"uplink-rx_bytes": 80,
		"uplink-rx_packets": 4,
		"uplink-tx_bytes": 20,
		"uplink-tx_packets": 1
	},
	"uptime": "61",
	"version": "1.0.0"
}
`)),
			ap: &AP{
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
				Stats: &APStats{
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
						ReceiveBytes:    80,
						ReceivePackets:  4,
						TransmitBytes:   20,
						TransmitPackets: 1,
					},
				},
				Uptime:  61 * time.Second,
				Version: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ap := new(AP)
			err := ap.UnmarshalJSON(tt.b)
			if want, got := errStr(tt.err), errStr(err); !strings.Contains(got, want) {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.ap, ap; !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected AP:\n- want: %+v\n-  got: %+v",
					want, got)
			}
		})
	}
}
