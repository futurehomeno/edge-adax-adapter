package model

import (
	"github.com/futurehomeno/fimpgo/fimptype"
)

// NetworkService is for export
type NetworkService struct {
}

// MakeInclusionReport makes inclusion report for player with id given in parameter
func (ns *NetworkService) MakeInclusionReport(id string, name string) fimptype.ThingInclusionReport {
	// var err error

	var deviceAddr, manufacturer string
	services := []fimptype.Service{}

	thermostatInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.setpoint.set",
		ValueType: "str_map",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.setpoint.report",
		ValueType: "str_map",
		Version:   "1",
	}, {
		Type:      "in",
		MsgType:   "cmd.setpoint.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "in",
		MsgType:   "cmd.mode.set",
		ValueType: "string",
		Version:   "1",
	}, {
		Type:      "in",
		MsgType:   "cmd.mode.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.mode.report",
		ValueType: "string",
		Version:   "1",
	}}

	sensorInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.sensor.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.sensor.report",
		ValueType: "float",
		Version:   "1",
	}}

	meterElecInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.meter.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.meter.report",
		ValueType: "float",
		Version:   "1",
	}}

	thermostatService := fimptype.Service{
		Name:    "thermostat",
		Alias:   "thermostat",
		Address: "/rt:dev/rn:adax/ad:1/sv:thermostat/ad:",
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_modes":     []string{"heat"},
			"sup_setpoints": []string{"heat"},
		},
		Interfaces: thermostatInterfaces,
	}

	tempSensorService := fimptype.Service{
		Name:    "sensor_temp",
		Alias:   "Temperature sensor",
		Address: "/rt:dev/rn:adax/ad:1/sv:sensor_temp/ad:",
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_units": []string{"C"},
		},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       sensorInterfaces,
	}

	meterElecService := fimptype.Service{
		Name:    "meter_elec",
		Alias:   "Meter Elec",
		Address: "/rt:dev/rn:adax/ad:1/sv:meter_elec/ad:",
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_units": []string{"kWh"},
		},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       meterElecInterfaces,
	}

	manufacturer = "adax"
	serviceAddress := id
	thermostatService.Address = thermostatService.Address + serviceAddress
	tempSensorService.Address = tempSensorService.Address + serviceAddress
	meterElecService.Address = meterElecService.Address + serviceAddress
	services = append(services, thermostatService, tempSensorService, meterElecService)
	deviceAddr = id
	powerSource := "ac"

	inclReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           deviceAddr,
		Type:              "",
		ProductHash:       manufacturer,
		CommTechnology:    "wifi",
		ProductName:       name,
		ManufacturerId:    manufacturer,
		DeviceId:          id,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"ch_0"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}

	return inclReport
}
