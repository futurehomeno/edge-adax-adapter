package router

import (
	"github.com/futurehomeno/edge-adax-adapter/model"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

func (fc *FromFimpRouter) handleMeterElecMessage(deviceID string, oldMsg *fimpgo.Message) {
	switch oldMsg.Payload.Type {
	case "cmd.meter.get_report":
		if err := fc.getStates(); err != nil {
			return
		}
		if err := fc.sendMeterReport(deviceID, oldMsg); err != nil {
			return
		}
	}
}

func (fc *FromFimpRouter) sendMeterReport(deviceID string, oldMsg *fimpgo.Message) error {
	_, _, device, err := fc.findHomeRoomAndDeviceFromDeviceID(deviceID)
	if err != nil {
		log.Error(err)

		return err
	}

	val := float64(device.PowerUsage.Energy) / 1000
	props := fimpgo.Props{}
	props["unit"] = "kWh"

	adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "meter_elec", ServiceAddress: deviceID}
	msg := fimpgo.NewMessage("evt.meter.report", "meter", fimpgo.VTypeFloat, val, props, nil, oldMsg.Payload)
	if err := fc.mqt.Publish(adr, msg); err != nil {
		log.Error(err)

		return err
	}
	return nil
}
