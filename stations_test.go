package unifi

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClientStations(t *testing.T) {
	const (
		wantSite     = "default"
		wantID       = "abcdef123457890"
		wantHostname = "somehost"
	)
	var (
		wantStationMAC = net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}
		wantIP         = net.IPv4(192, 168, 1, 2)
		wantMAC        = net.HardwareAddr{0xab, 0xad, 0x1d, 0xea, 0xab, 0xad}
	)

	zeroUNIX := time.Unix(0, 0)

	wantStation := &Station{
		ID:              wantID,
		APMAC:           wantStationMAC,
		AssociationTime: zeroUNIX,
		FirstSeen:       zeroUNIX,
		Hostname:        wantHostname,
		IP:              wantIP,
		LastSeen:        zeroUNIX,
		MAC:             wantMAC,
		SiteID:          wantSite,
		Stats:           &StationStats{},
	}

	v := struct {
		Stations []station `json:"data"`
	}{
		Stations: []station{{
			ID:       wantID,
			ApMac:    wantStationMAC.String(),
			Hostname: wantHostname,
			IP:       wantIP.String(),
			Mac:      wantMAC.String(),
			SiteID:   wantSite,
		}},
	}

	c, done := testClient(t, testHandler(
		t,
		http.MethodGet,
		fmt.Sprintf("/api/s/%s/stat/sta", wantSite),
		nil,
		v,
	))
	defer done()

	stations, err := c.Stations(wantSite)
	if err != nil {
		t.Fatalf("unexpected error from Client.Stations: %v", err)
	}

	if want, got := 1, len(stations); want != got {
		t.Fatalf("unexpected number of Stations:\n- want: %d\n-  got: %d",
			want, got)
	}

	if want, got := wantStation, stations[0]; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Station:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestStationUnmarshalJSON(t *testing.T) {
	zeroUNIX := time.Unix(0, 0)

	var tests = []struct {
		desc string
		b    []byte
		s    *Station
		err  error
	}{
		{
			desc: "invalid JSON",
			b:    []byte(`<>`),
			err:  errors.New("invalid character"),
		},
		{
			desc: "invalid AP MAC",
			b:    []byte(`{"ap_mac":"foo"}`),
			err:  errors.New("invalid MAC address"),
		},
		{
			desc: "invalid IP",
			b:    []byte(`{"ap_mac":"de:ad:be:ef:de:ad","ip":"foo"}`),
			err:  errors.New("failed to parse station IP"),
		},
		{
			desc: "invalid MAC",
			b:    []byte(`{"ap_mac":"de:ad:be:ef:de:ad","ip":"192.168.1.2","mac":"foo"}`),
			err:  errors.New("invalid MAC address"),
		},
		{
			desc: "OK",
			b: bytes.TrimSpace([]byte(`
{
	"_id": "abcdef1234567890",
	"ap_mac": "ab:ad:1d:ea:ab:ad",
	"channel": 1,
	"hostname": "somehost",
	"ip": "192.168.1.2",
	"mac": "de:ad:be:ef:de:ad",
	"roam_count": 1,
	"site_id": "somesite",
	"rx_bytes": 80,
	"rx_packets": 4,
	"rx_rate": 1024,
	"tx_bytes": 20,
	"tx_packets": 1,
	"tx_power": 10,
	"tx_rate": 1024,
	"uptime": 10,
	"user_id": "someuser"
}
`)),
			s: &Station{
				ID:              "abcdef1234567890",
				APMAC:           net.HardwareAddr{0xab, 0xad, 0x1d, 0xea, 0xab, 0xad},
				AssociationTime: zeroUNIX,
				Channel:         1,
				FirstSeen:       zeroUNIX,
				Hostname:        "somehost",
				IP:              net.IPv4(192, 168, 1, 2),
				LastSeen:        zeroUNIX,
				MAC:             net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad},
				RoamCount:       1,
				SiteID:          "somesite",
				Stats: &StationStats{
					ReceiveBytes:    80,
					ReceivePackets:  4,
					ReceiveRate:     1024,
					TransmitBytes:   20,
					TransmitPackets: 1,
					TransmitPower:   10,
					TransmitRate:    1024,
				},
				Uptime: 10 * time.Second,
				UserID: "someuser",
			},
		},
	}

	for i, tt := range tests {
		t.Logf("[%02d] test %q", i, tt.desc)

		s := new(Station)
		err := s.UnmarshalJSON(tt.b)
		if want, got := errStr(tt.err), errStr(err); !strings.Contains(got, want) {
			t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
				want, got)
		}
		if err != nil {
			continue
		}

		if want, got := tt.s, s; !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected Station:\n- want: %+v\n-  got: %+v",
				want, got)
		}
	}
}
