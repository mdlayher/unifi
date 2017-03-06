package unifi

import (
	"encoding/json"
	"fmt"
	"net"
)

// A Device is a Ubiquiti UniFi device, such as an access point or switch.
type Device interface {
	Type() DeviceType
}

// A DeviceType indicates the type of a Device, such as an AP or Switch.
type DeviceType int

// Possible DeviceType values.
const (
	DeviceTypeUnknown = iota
	DeviceTypeAP
	DeviceTypeSwitch
)

// Possible device type identifiers, used to determine a Device's type.
const (
	// UniFi access point.
	deviceUAP = "uap"
)

// Devices returns all of the Devices for a specified site name.
func (c *Client) Devices(siteName string) ([]Device, error) {
	req, err := c.newRequest(
		"GET",
		fmt.Sprintf("/api/s/%s/stat/device", siteName),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var v struct {
		Data []json.RawMessage `json:"data"`
	}

	if _, err := c.do(req, &v); err != nil {
		return nil, err
	}

	var typ struct {
		Type string `json:"type"`
	}

	devices := make([]Device, 0, len(v.Data))
	for _, data := range v.Data {
		if err := json.Unmarshal(data, &typ); err != nil {
			return nil, err
		}

		var d Device
		switch typ.Type {
		case deviceUAP:
			d = new(AP)
		default:
			continue
		}

		if err := json.Unmarshal(data, d); err != nil {
			return nil, err
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// A NIC is a wired ethernet network interface on a device.
type NIC struct {
	MAC  net.HardwareAddr
	Name string
}

// WiredStats contains wired device network activity statistics.
type WiredStats struct {
	ReceiveBytes    int64
	ReceivePackets  int64
	TransmitBytes   int64
	TransmitPackets int64
}
