package router

import (
	"github.com/futurehomeno/edge-adax-adapter/model"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

func (fc *FromFimpRouter) handleSensorTempMessage(deviceID string, oldMsg *fimpgo.Message) {
	switch oldMsg.Payload.Type {
	case "cmd.sensor.get_report":
		if err := fc.getStates(); err != nil {
			return
		}
		if err := fc.sendTempReport(deviceID, oldMsg); err != nil {
			return
		}
	}
}

func (fc *FromFimpRouter) sendTempReport(deviceID string, oldMsg *fimpgo.Message) error {
	_, room, _, err := fc.findHomeRoomAndDeviceFromDeviceID(deviceID)
	if err != nil {
		log.Error(err)

		return err
	}

	val := room.Temperature / 100
	props := fimpgo.Props{}
	props["unit"] = "C"

	adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_temp", ServiceAddress: deviceID}
	msg := fimpgo.NewMessage("evt.sensor.report", "sensor", fimpgo.VTypeFloat, val, props, nil, oldMsg.Payload)
	if err := fc.mqt.Publish(adr, msg); err != nil {
		log.Error(err)

		return err
	}
	return nil
}
