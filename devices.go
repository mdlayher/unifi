package unifi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"
)

// Devices returns all of the Devices for a specified site name.
func (c *Client) Devices(siteName string) ([]*Device, error) {
	var v struct {
		Devices []*Device `json:"data"`
	}

	req, err := c.newRequest(
		"GET",
		fmt.Sprintf("/api/s/%s/stat/device", siteName),
		nil,
	)
	if err != nil {
		return nil, err
	}

	_, err = c.do(req, &v)
	return v.Devices, err
}

// A Device is a Ubiquiti UniFi device, such as a UniFi access point.
type Device struct {
	ID        string
	Adopted   bool
	InformIP  net.IP
	InformURL *url.URL
	Model     string
	Name      string
	NICs      []*NIC
	Radios    []*Radio
	Serial    string
	SiteID    string
	Stats     *DeviceStats
	Uptime    time.Duration
	Version   string

	// TODO(mdlayher): add more fields from unexported device type
}

// A Radio is a wireless radio, attached to a Device.
type Radio struct {
	BuiltInAntenna     bool
	BuiltInAntennaGain int
	MaxTXPower         int
	MinTXPower         int
	Name               string
	Radio              string
	Stats              *RadioStationsStats
}

// RadioStationsStats contains Station statistics for a Radio.
type RadioStationsStats struct {
	NumberStations      int
	NumberGuestStations int
	NumberUserStations  int
}

// A NIC is a wired ethernet network interface, attached to a Device.
type NIC struct {
	MAC  net.HardwareAddr
	Name string
}

// DeviceStats contains device network activity statistics.
type DeviceStats struct {
	TotalBytes float64
	All        *WirelessStats
	Guest      *WirelessStats
	User       *WirelessStats
	Uplink     *WiredStats
	System     *SystemStats
}

func (s *DeviceStats) String() string {
	return fmt.Sprintf("%v", *s)
}

type SystemStats struct {
	CpuPercentage float64
	MemPercentage float64
	Uptime        int
	LoadAvg1      float64
	LoadAvg15     float64
	LoadAvg5      float64
	MemBuffer     int64
	MemTotal      int64
	MemUsed       int64
}

// WirelessStats contains wireless device network activity statistics.
type WirelessStats struct {
	ReceiveBytes    float64
	ReceivePackets  float64
	TransmitBytes   float64
	TransmitDropped float64
	TransmitPackets float64
}

func (s *WirelessStats) String() string {
	return fmt.Sprintf("%v", *s)
}

// WiredStats contains wired device network activity statistics.
type WiredStats struct {
	ReceiveBytes    float64
	ReceivePackets  float64
	TransmitBytes   float64
	TransmitPackets float64
}

func (s *WiredStats) String() string {
	return fmt.Sprintf("%v", *s)
}

const (
	radioNA = "na"
	radioNG = "ng"

	radio5GHz  = "5GHz"
	radio24GHz = "2.4GHz"
)

// UnmarshalJSON unmarshals the raw JSON representation of a Device.
func (d *Device) UnmarshalJSON(b []byte) error {
	var dev device
	if err := json.Unmarshal(b, &dev); err != nil {
		return err
	}

	informIP := net.ParseIP(dev.InformIP)
	if informIP == nil {
		return fmt.Errorf("failed to parse inform IP: %v", dev.InformIP)
	}

	informURL, err := url.Parse(dev.InformURL)
	if err != nil {
		return err
	}

	nics := make([]*NIC, 0, len(dev.EthernetTable))
	for _, et := range dev.EthernetTable {
		mac, err := net.ParseMAC(et.MAC)
		if err != nil {
			return err
		}

		nics = append(nics, &NIC{
			MAC:  mac,
			Name: et.Name,
		})
	}

	radios := make([]*Radio, 0, len(dev.RadioTable))
	for _, rt := range dev.RadioTable {
		r := &Radio{
			BuiltInAntenna:     rt.BuiltinAntenna,
			BuiltInAntennaGain: rt.BuiltinAntGain,
			MaxTXPower:         rt.MaxTXPower,
			MinTXPower:         rt.MinTXPower,
			Name:               rt.Name,
		}

		for _, v := range dev.RadioTableStats {
			if v.Name == rt.Name {
				r.Stats = &RadioStationsStats{
					NumberStations:      v.NumSta,
					NumberUserStations:  v.UserNumSta,
					NumberGuestStations: v.GuestNumSta,
				}
			}
		}

		switch rt.Radio {
		case radioNA:
			r.Radio = radio5GHz
		case radioNG:
			r.Radio = radio24GHz
		}

		radios = append(radios, r)
	}

	*d = Device{
		ID:        dev.ID,
		Adopted:   dev.Adopted,
		InformIP:  informIP,
		InformURL: informURL,
		Model:     dev.Model,
		Name:      dev.Name,
		NICs:      nics,
		Radios:    radios,
		Serial:    dev.Serial,
		SiteID:    dev.SiteID,
		Uptime:    time.Duration(time.Duration(dev.Uptime) * time.Second),
		Version:   dev.Version,
		Stats: &DeviceStats{
			TotalBytes: dev.Stat.Bytes,
			All: &WirelessStats{
				ReceiveBytes:    dev.Stat.RxBytes,
				ReceivePackets:  dev.Stat.RxPackets,
				TransmitBytes:   dev.Stat.TxBytes,
				TransmitDropped: dev.Stat.TxDropped,
				TransmitPackets: dev.Stat.TxPackets,
			},
			User: &WirelessStats{
				ReceiveBytes:    dev.Stat.UserRxBytes,
				ReceivePackets:  dev.Stat.UserRxPackets,
				TransmitBytes:   dev.Stat.UserTxBytes,
				TransmitDropped: dev.Stat.UserTxDropped,
				TransmitPackets: dev.Stat.UserTxPackets,
			},
			Guest: &WirelessStats{
				ReceiveBytes:    dev.Stat.GuestRxBytes,
				ReceivePackets:  dev.Stat.GuestRxPackets,
				TransmitBytes:   dev.Stat.GuestTxBytes,
				TransmitDropped: dev.Stat.GuestTxDropped,
				TransmitPackets: dev.Stat.GuestTxPackets,
			},
			Uplink: &WiredStats{
				ReceiveBytes:    dev.Uplink.RxBytes,
				ReceivePackets:  dev.Uplink.RxPackets,
				TransmitBytes:   dev.Uplink.TxBytes,
				TransmitPackets: dev.Uplink.TxPackets,
			},
			System: &SystemStats{
				Uptime:        dev.SystemStats.Uptime,
				CpuPercentage: dev.SystemStats.CpuPercentage,
				MemPercentage: dev.SystemStats.MemPercentage,
				LoadAvg1:      dev.SysStats.LoadAvg1,
				LoadAvg15:     dev.SysStats.LoadAvg15,
				LoadAvg5:      dev.SysStats.LoadAvg5,
				MemBuffer:     dev.SysStats.MemBuffer,
				MemTotal:      dev.SysStats.MemTotal,
				MemUsed:       dev.SysStats.MemUsed,
			},
		},
	}

	return nil
}

// A device is the raw structure of a Device returned from the UniFi Controller
// API.
type device struct {
	// TODO(mdlayher): give all fields appropriate names and data types.
	ID            string  `json:"_id"`
	Adopted       bool    `json:"adopted"`
	Bytes         float64 `json:"bytes"`
	ConfigVersion string  `json:"cfgversion"`
	ConfigNetwork struct {
		IP   string `json:"ip"`
		Type string `json:"type"`
	} `json:"config_network"`
	DeviceID      string `json:"device_id"`
	EthernetTable []struct {
		MAC     string `json:"mac"`
		Name    string `json:"name"`
		NumPort int    `json:"num_port"`
	} `json:"ethernet_table"`
	GuestNumSta int    `json:"guest-num_sta"`
	HasSpeaker  bool   `json:"has_speaker"`
	InformIP    string `json:"inform_ip"`
	InformURL   string `json:"inform_url"`
	IP          string `json:"ip"`
	LastSeen    int    `json:"last_seen"`
	MAC         string `json:"mac"`
	Model       string `json:"model"`
	Name        string `json:"name"`
	NumSta      int    `json:"num_sta"`
	RadioNg     struct {
		BuiltInAntennaGain int    `json:"builtin_ant_gain"`
		BuiltInAntenna     bool   `json:"builtin_antenna"`
		MaxTXPower         int    `json:"max_txpower"`
		MinTXPower         int    `json:"min_txpower"`
		Name               string `json:"name"`
		Radio              string `json:"radio"`
	} `json:"radio_ng"`
	RadioTable []struct {
		BuiltinAntGain int    `json:"builtin_ant_gain"`
		BuiltinAntenna bool   `json:"builtin_antenna"`
		MaxTXPower     int    `json:"max_txpower"`
		MinTXPower     int    `json:"min_txpower"`
		Name           string `json:"name"`
		Radio          string `json:"radio"`
	} `json:"radio_table"`
	RadioTableStats []struct {
		AstBeXmit   int         `json:"ast_be_xmit"`
		AstCst      int         `json:"ast_cst"`
		AstTxto     interface{} `json:"ast_txto"`
		Channel     int         `json:"channel"`
		CuSelfRx    int         `json:"cu_self_rx"`
		CuSelfTx    int         `json:"cu_self_tx"`
		CuTotal     int         `json:"cu_total"`
		Extchannel  int         `json:"extchannel"`
		Gain        int         `json:"gain"`
		GuestNumSta int         `json:"guest-num_sta"`
		Name        string      `json:"name"`
		NumSta      int         `json:"num_sta"`
		Radio       string      `json:"radio"`
		State       string      `json:"state"`
		TxPackets   int         `json:"tx_packets"`
		TxPower     int         `json:"tx_power"`
		TxRetries   int         `json:"tx_retries"`
		UserNumSta  int         `json:"user-num_sta"`
	} `json:"radio_table_stats"`
	RxBytes float64 `json:"rx_bytes"`
	Serial  string  `json:"serial,omitempty"`
	SiteID  string  `json:"site_id"`
	Stat    struct {
		Bytes          float64 `json:"bytes"`
		GuestRxBytes   float64 `json:"guest-rx_bytes"`
		GuestRxPackets float64 `json:"guest-rx_packets"`
		GuestTxBytes   float64 `json:"guest-tx_bytes"`
		GuestTxDropped float64 `json:"guest-tx_dropped"`
		GuestTxPackets float64 `json:"guest-tx_packets"`
		Mac            string  `json:"mac"`
		RxBytes        float64 `json:"rx_bytes"`
		RxPackets      float64 `json:"rx_packets"`
		TxBytes        float64 `json:"tx_bytes"`
		TxDropped      float64 `json:"tx_dropped"`
		TxPackets      float64 `json:"tx_packets"`
		UserRxBytes    float64 `json:"user-rx_bytes"`
		UserRxPackets  float64 `json:"user-rx_packets"`
		UserTxBytes    float64 `json:"user-tx_bytes"`
		UserTxDropped  float64 `json:"user-tx_dropped"`
		UserTxPackets  float64 `json:"user-tx_packets"`
	} `json:"stat"`
	Uplink struct {
		RxBytes   float64 `json:"rx_bytes"`
		RxPackets float64 `json:"rx_packets"`
		RxErrors  float64 `json:"rx_errors"`
		TxBytes   float64 `json:"tx_bytes"`
		TxPackets float64 `json:"tx_packets"`
		TxErrors  float64 `json:"tx_errors"`
		Type      string  `json:"type"`
	} `json:"uplink"`
	State         int           `json:"state"`
	TxBytes       float64       `json:"tx_bytes"`
	Type          string        `json:"type"`
	UplinkTable   []interface{} `json:"uplink_table"`
	Uptime        int           `json:"uptime"`
	UserNumSta    int           `json:"user-num_sta"`
	Version       string        `json:"version"`
	VwireEnabled  bool          `json:"vwireEnabled"`
	VwireTable    []interface{} `json:"vwire_table"`
	WlangroupIDNg string        `json:"wlangroup_id_ng"`
	XAuthkey      string        `json:"x_authkey"`
	XFingerprint  string        `json:"x_fingerprint"`
	XVwirekey     string        `json:"x_vwirekey"`
	SystemStats   struct {
		CpuPercentage float64 `json:"cpu,string"`
		MemPercentage float64 `json:"mem,string"`
		Uptime        int     `json:"uptime,string"`
	} `json:"system-stats"`
	SysStats struct {
		LoadAvg1  float64 `json:"loadavg_1,string"`
		LoadAvg15 float64 `json:"loadavg_15,string"`
		LoadAvg5  float64 `json:"loadavg_5,string"`
		MemBuffer int64   `json:"mem_buffer"`
		MemTotal  int64   `json:"mem_total"`
		MemUsed   int64   `json:"mem_used"`
	} `json:"sys_stats"`
}
