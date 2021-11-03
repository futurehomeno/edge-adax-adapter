package router

import (
	"fmt"
	"strconv"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/adax/model"
)

func (fc *FromFimpRouter) handleThermostatMessage(deviceID string, oldMsg *fimpgo.Message) {
	switch oldMsg.Payload.Type {
	case "cmd.setpoint.set":
		if err := fc.handleSetpointSet(deviceID, oldMsg); err != nil {
			return
		}

	case "cmd.setpoint.get_report":
		if err := fc.handleSetpointGet(deviceID, oldMsg); err != nil {
			return
		}

	case "cmd.mode.set":
		// Do we need this? Will/should always be heat

	case "cmd.mode.get_report":
		val := "heat"

		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: deviceID}
		msg := fimpgo.NewMessage("evt.mode.report", "thermostat", fimpgo.VTypeString, val, nil, nil, oldMsg.Payload)
		fc.mqt.Publish(adr, msg)
	}
	return

}

func (fc *FromFimpRouter) handleSetpointSet(deviceID string, oldMsg *fimpgo.Message) error {
	val, _ := oldMsg.Payload.GetStrMapValue()
	newTemp, err := strconv.ParseFloat(val["temp"], 32)
	if err != nil {
		log.Error("Can't convert newtemp to float")
		return err
	}

	home, room, _, err := fc.findHomeRoomAndDeviceFromDeviceID(deviceID)
	if err != nil {
		log.Error(err)
		return err
	}

	if err := fc.setTemperature(home.ID, room.ID, newTemp); err != nil {
		return err
	}

	if err := fc.sendSetpointReport(deviceID, newTemp, nil); err != nil {
		return err
	}
	return nil
}

func (fc *FromFimpRouter) handleSetpointGet(deviceID string, oldMsg *fimpgo.Message) error {
	if err := fc.getStates(); err != nil {
		return err
	}

	_, room, _, err := fc.findHomeRoomAndDeviceFromDeviceID(deviceID)
	if err != nil {
		log.Error(err)
		return err
	}

	setpointTemp := float64(room.TargetTemperature / 100)
	if err := fc.sendSetpointReport(deviceID, setpointTemp, oldMsg); err != nil {
		return err
	}
	return nil
}

func (fc *FromFimpRouter) sendSetpointReport(deviceID string, setpointTemp float64, oldMsg *fimpgo.Message) error {
	val := map[string]interface{}{
		"type": "heat",
		"temp": fmt.Sprintf("%f", setpointTemp),
		"unit": "C",
	}

	var reqMessage *fimpgo.FimpMessage
	if oldMsg == nil {
		reqMessage = nil
	} else {
		reqMessage = oldMsg.Payload
	}

	adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: deviceID}
	msg := fimpgo.NewMessage("evt.setpoint.report", "thermostat", fimpgo.VTypeStrMap, val, nil, nil, reqMessage)
	fc.mqt.Publish(adr, msg)
	return nil
}
