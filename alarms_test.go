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

func TestClientAlarms(t *testing.T) {
	const (
		wantSite    = "default"
		wantID      = "abcdef123457890"
		wantAPName  = "ap001"
		wantMessage = "ap001 was disconnected"
	)
	var (
		wantAPMAC    = net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}
		wantDateTime = time.Date(2016, time.January, 01, 0, 0, 0, 0, time.UTC)
	)

	wantAlarm := &Alarm{
		ID:       wantID,
		APMAC:    wantAPMAC,
		APName:   wantAPName,
		DateTime: wantDateTime,
		Message:  wantMessage,
	}

	v := struct {
		Alarms []alarm `json:"data"`
	}{
		Alarms: []alarm{{
			ID:       wantID,
			AP:       wantAPMAC.String(),
			APName:   wantAPName,
			DateTime: wantDateTime.Format(time.RFC3339),
			Msg:      wantMessage,
		}},
	}

	c, done := testClient(t, testHandler(
		t,
		http.MethodGet,
		fmt.Sprintf("/api/s/%s/list/alarm", wantSite),
		nil,
		v,
	))
	defer done()

	alarms, err := c.Alarms(wantSite)
	if err != nil {
		t.Fatalf("unexpected error from Client.Alarms: %v", err)
	}

	if want, got := 1, len(alarms); want != got {
		t.Fatalf("unexpected number of Alarms:\n- want: %d\n-  got: %d",
			want, got)
	}

	if want, got := wantAlarm, alarms[0]; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Alarm:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestAlarmUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		desc string
		b    []byte
		a    *Alarm
		err  error
	}{
		{
			desc: "invalid JSON",
			b:    []byte(`<>`),
			err:  errors.New("invalid character"),
		},
		{
			desc: "invalid APMAC",
			b:    []byte(`{"ap":"foo"}`),
			err:  errors.New("invalid MAC address"),
		},
		{
			desc: "invalid DateTime",
			b:    []byte(`{"ap":"de:ad:be:ef:de:ad","datetime":"foo"}`),
			err:  errors.New("parsing time"),
		},
		{
			desc: "OK",
			b: bytes.TrimSpace([]byte(`
{
	"_id": "abcdef1234567890",
	"ap": "de:ad:be:ef:de:ad",
	"ap_name": "ap001",
	"archived": false,
	"datetime": "2016-01-01T00:00:00Z"
	"key": "EVT AP Lost Contact",
	"msg": "ap001 was disconnected"
	"site_id": "default",
	"subsystem": "wlan"
}
`)),
			a: &Alarm{
				ID:        "abcdef1234567890",
				APMAC:     net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad},
				APName:    "ap001",
				DateTime:  time.Date(2016, time.January, 01, 0, 0, 0, 0, time.UTC),
				Key:       "EVT AP Lost Contact",
				Message:   "ap001 was disconnected",
				SiteID:    "default",
				Subsystem: "wlan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			a := new(Alarm)
			err := a.UnmarshalJSON(tt.b)
			if want, got := errStr(tt.err), errStr(err); !strings.Contains(got, want) {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.a, a; !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected Alarm:\n- want: %+v\n-  got: %+v",
					want, got)
			}
		})
	}
}
