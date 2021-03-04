package router

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/adax/adax-api"
	"github.com/thingsplex/adax/model"
)

type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceId   string
	appLifecycle *model.Lifecycle
	configs      *model.Configs
	states       *model.States
	client       *adax.Client
	env          string
}

// NewFromFimpRouter ...
func NewFromFimpRouter(mqt *fimpgo.MqttTransport, appLifecycle *model.Lifecycle, configs *model.Configs, states *model.States, client *adax.Client) *FromFimpRouter {
	fc := FromFimpRouter{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: mqt, appLifecycle: appLifecycle, configs: configs, states: states, client: client}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	hubInfo, err := utils.NewHubUtils().GetHubInfo()
	if err == nil && hubInfo != nil {
		fc.env = hubInfo.Environment
	} else {
		fc.env = utils.EnvProd
	}
	return &fc
}

// Start ...
func (fc *FromFimpRouter) Start() {

	if err := fc.mqt.Subscribe(fmt.Sprintf("pt:j1/mt:cmd/rt:dev/rn:%s/ad:1/#", model.ServiceName)); err != nil {
		log.Error(err)
	}
	if err := fc.mqt.Subscribe(fmt.Sprintf("pt:j1/mt:cmd/rt:ad/rn:%s/ad:1", model.ServiceName)); err != nil {
		log.Error(err)
	}

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	hr := adax.HomesAndRooms{}
	state := adax.State{}
	ns := model.NetworkService{}

	log.Debug("New fimp msg . cmd = ", newMsg.Payload.Type)
	addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	// ns := model.NetworkService{}
	switch newMsg.Payload.Service {

	case "thermostat":
		addr = strings.Replace(addr, "l", "", 1)
		deviceID, err := strconv.Atoi(addr)
		if err != nil {
			log.Error("Can't convert deviceID to int")
		}
		switch newMsg.Payload.Type {
		case "cmd.setpoint.set":
			val, _ := newMsg.Payload.GetStrMapValue()
			newTemp, err := strconv.ParseFloat(val["temp"], 32)
			if err != nil {
				log.Error("Can't convert newtemp to float")
				return
			}
			for _, homes := range fc.states.HomesAndRooms.Users[0].Homes {
				for _, rooms := range homes.Rooms {
					for _, device := range rooms.Devices {

						if deviceID == device.ID {
							err = state.SetTemperature(fc.configs.User, homes.ID, rooms.ID, newTemp, fc.configs.AccessToken)
							if err != nil {
								log.Error(err)
								return
							}
							adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: addr}
							msg := fimpgo.NewMessage("evt.setpoint.report", "thermostat", fimpgo.VTypeStrMap, val, nil, nil, newMsg.Payload)
							fc.mqt.Publish(adr, msg)
						}
					}
				}
			}

		case "cmd.setpoint.get_report":
			fc.states.States = nil
			var err error
			fc.states.States, err = state.GetStates(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			}
			for _, homes := range fc.states.States.Users[0].Homes {
				for _, rooms := range homes.Rooms {
					for _, device := range rooms.Devices {
						if deviceID == device.ID {
							setpointTemp := rooms.TargetTemperature / 100
							if setpointTemp != 0 {
								val := map[string]interface{}{
									"type": "heat",
									"temp": strconv.Itoa(setpointTemp),
									"unit": "C",
								}
								adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: addr}
								msg := fimpgo.NewMessage("evt.setpoint.report", "thermostat", fimpgo.VTypeStrMap, val, nil, nil, newMsg.Payload)
								fc.mqt.Publish(adr, msg)
							}
						}
					}
				}
			}

		case "cmd.mode.set":
			// Do we need this? Will/should always be heat

		case "cmd.mode.get_report":
			val := "heat"

			adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: addr}
			msg := fimpgo.NewMessage("evt.mode.report", "thermostat", fimpgo.VTypeString, val, nil, nil, newMsg.Payload)
			fc.mqt.Publish(adr, msg)
		}

	case "sensor_temp":

		addr = strings.Replace(addr, "l", "", 1)
		deviceID, err := strconv.Atoi(addr)
		if err != nil {
			log.Error("Can't convert deviceID to int")
		}
		switch newMsg.Payload.Type {

		case "cmd.sensor.get_report":
			fc.states.States = nil
			var err error
			fc.states.States, err = state.GetStates(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			}
			for _, homes := range fc.states.States.Users[0].Homes {
				for _, rooms := range homes.Rooms {
					for _, device := range rooms.Devices {
						if deviceID == device.ID {
							val := rooms.Temperature / 100
							props := fimpgo.Props{}
							props["unit"] = "C"

							adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_temp", ServiceAddress: addr}
							msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", fimpgo.VTypeFloat, val, props, nil, newMsg.Payload)
							fc.mqt.Publish(adr, msg)
						}
					}
				}
			}
		}

	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {
		case "cmd.auth.set_tokens":
			authReq := model.SetTokens{}
			err := newMsg.Payload.GetObjectValue(&authReq)
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			status := model.AuthStatus{
				Status:    model.AuthStateAuthenticated,
				ErrorText: "",
				ErrorCode: "",
			}
			if authReq.AccessToken != "" && authReq.RefreshToken != "" {
				fc.client.SetTokens(authReq.AccessToken, authReq.RefreshToken)

				// After getting first access token, use this to get users
				fc.configs.User, err = fc.client.GetUsers(authReq.AccessToken)
				if err != nil {
					log.Error(err)
					return
				}
				log.Debug("Users: ", fc.configs.User)

				// After getting users, get "code"
				fc.configs.Code, err = fc.client.GetCode()
				if err != nil {
					log.Error(err)
					return
				}
				log.Debug("Code: ", fc.configs.Code)

				// After getting code, get new access/refresh tokens
				fc.configs.AccessToken, fc.configs.RefreshToken, err = fc.client.GetTokens(fc.configs.Code)
				log.Debug("Access token: ", fc.configs.AccessToken)
				log.Debug("Refresh token: ", fc.configs.RefreshToken)

				if err != nil {
					log.Error(err)
					return
				}
				fc.appLifecycle.SetConfigState(model.ConfigStateConfigured)
				fc.appLifecycle.SetAuthState(model.AuthStateAuthenticated)
				fc.appLifecycle.SetConnectionState(model.ConnStateConnected)
				fc.appLifecycle.SetAppState(model.AppStateRunning, nil)

			} else {
				status.Status = "ERROR"
				status.ErrorText = "Accesstoken or refreshtoken empty"
			}
			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				if err := fc.mqt.Publish(adr, msg); err != nil {
					log.Error(err)
				}
			}

			// Get homes and rooms
			fc.states.HomesAndRooms, err = hr.GetHomesAndRooms(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			} else {
				log.Info("New tokens set successfully")
				log.Info("HomesAndRooms: ", fc.states.HomesAndRooms)
			}

			// Get states
			fc.states.States, err = state.GetStates(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			} else {
				log.Info("New states set successfully")
				log.Info("States: ", fc.states.States)
				// Send inclusion report for all devices
				for _, home := range fc.states.HomesAndRooms.Users[0].Homes {
					for _, room := range home.Rooms {
						for _, device := range room.Devices {
							// Include here
							// fc.configs.DeviceCollection = append(fc.configs.DeviceCollection, device)
							inclReport := ns.MakeInclusionReport(strconv.Itoa(device.ID), device.Name)

							msg := fimpgo.NewMessage("evt.thing.inclusion_report", "adax", fimpgo.VTypeObject, inclReport, nil, nil, nil)
							adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
							fc.mqt.Publish(&adr, msg)
						}
					}
				}
			}

			if err := fc.configs.SaveToFile(); err != nil {
				log.Error("<frouter> Can't save configurations. Err: ", err)
			}
		case "cmd.auth.logout":
			fc.configs.AccessToken = ""
			fc.configs.Code = ""
			fc.configs.RefreshToken = ""
			fc.configs.User = 0
			fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
			fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			fc.appLifecycle.SetConnectionState(model.ConnStateDisconnected)
			fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
			for _, home := range fc.states.HomesAndRooms.Users[0].Homes {
				for _, room := range home.Rooms {
					for _, device := range room.Devices {
						log.Info("Excluding device: ", device.ID)
						exclVal := map[string]interface{}{
							"address": strconv.Itoa(device.ID),
						}
						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
						msg := fimpgo.NewMessage("evt.thing.exclusion_report", "adax", fimpgo.VTypeObject, exclVal, nil, nil, newMsg.Payload)
						fc.mqt.Publish(adr, msg)
					}
				}
			}
			fc.configs.LoadDefaults()
			fc.states.LoadDefaults()
			logoutVal := map[string]interface{}{
				"errors":  nil,
				"success": true,
			}
			msg := fimpgo.NewMessage("evt.pd7.response", "vinculum", fimpgo.VTypeObject, logoutVal, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				log.Error("Could not respond to wanted request")
			}

		case "cmd.app.get_manifest":
			mode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect request format ")
				return
			}
			manifest := model.NewManifest()
			err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
			if err != nil {
				log.Error("Failed to load manifest file .Error :", err.Error())
				return
			}
			if mode == "manifest_state" {
				manifest.AppState = *fc.appLifecycle.GetAllStates()
				fc.configs.ConnectionState = string(fc.appLifecycle.ConnectionState())
				fc.configs.Errors = fc.appLifecycle.LastError()
				manifest.ConfigState = fc.configs
			}
			if errConf := manifest.GetAppConfig("errors"); errConf != nil {
				if fc.configs.Errors == "" {
					errConf.Hidden = true
				} else {
					errConf.Hidden = false
				}
			}

			syncButton := manifest.GetButton("sync")
			pollTimeBlock := manifest.GetUIBlock("poll_time_min")
			if fc.appLifecycle.ConnectionState() == model.ConnStateConnected {
				syncButton.Hidden = false
				pollTimeBlock.Hidden = false
			} else {
				syncButton.Hidden = true
				pollTimeBlock.Hidden = true
			}

			if fc.env == utils.EnvBeta {
				manifest.Auth.AuthEndpoint = "https://partners-beta.futurehome.io/api/edge/proxy/custom/auth-code"
				manifest.Auth.RedirectURL = "https://app-static-beta.futurehome.io/playground_oauth_callback"
			} else {
				manifest.Auth.AuthEndpoint = "https://partners.futurehome.io/api/edge/proxy/custom/auth-code"
				manifest.Auth.RedirectURL = "https://app-static.futurehome.io/playground_oauth_callback"
			}

			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_state":
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.system.sync":
			var err error
			fc.states.HomesAndRooms, err = hr.GetHomesAndRooms(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			} else {
				log.Info("HomesAndRooms: ", fc.states.HomesAndRooms)
			}

			// Get states
			fc.states.States, err = state.GetStates(fc.configs.User, fc.configs.AccessToken)
			if err != nil {
				log.Error("error: ", err)
			} else {
				log.Info("States: ", fc.states.States)
				// Send inclusion report for all devices
				for _, home := range fc.states.HomesAndRooms.Users[0].Homes {
					for _, room := range home.Rooms {
						for _, device := range room.Devices {
							// Include here
							// fc.configs.DeviceCollection = append(fc.configs.DeviceCollection, device)
							inclReport := ns.MakeInclusionReport(strconv.Itoa(device.ID), device.Name)

							msg := fimpgo.NewMessage("evt.thing.inclusion_report", "adax", fimpgo.VTypeObject, inclReport, nil, nil, nil)
							adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
							fc.mqt.Publish(&adr, msg)
						}
					}
				}
				val2 := map[string]interface{}{
					"errors":  nil,
					"success": true,
				}
				msg := fimpgo.NewMessage("evt.pd7.response", "vinculum", fimpgo.VTypeObject, val2, nil, nil, newMsg.Payload)
				if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
					log.Error("Could not respond to wanted request")
				}
			}

		case "cmd.config.get_extended_report":

			msg := fimpgo.NewMessage("evt.config.extended_report", model.ServiceName, fimpgo.VTypeObject, fc.configs, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.extended_set":
			conf := model.Configs{}
			err := newMsg.Payload.GetObjectValue(&conf)
			if err != nil {
				// TODO: This is an example . Add your logic here or remove
				log.Error("Can't parse configuration object")
				return
			}
			pollTimeMin := conf.PollTimeMin
			_, err = strconv.Atoi(pollTimeMin)

			if err != nil {
				log.Error(fmt.Sprintf("%q is not a number or contains illegal symbols.", pollTimeMin))
			} else {
				fc.configs.PollTimeMin = pollTimeMin
				fc.configs.SaveToFile()
				log.Debugf("App reconfigured . New parameters : %v", fc.configs)
			}

			configReport := model.ConfigReport{
				OpStatus: "ok",
				AppState: *fc.appLifecycle.GetAllStates(),
			}
			msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.log.set_level":
			// Configure log level
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.system.reconnect":
			// This is optional operation.
			fc.appLifecycle.PublishEvent(model.EventConfigured, "from-fimp-router", nil)
			//val := map[string]string{"status":status,"error":errStr}
			val := model.ButtonActionResponse{
				Operation:       "cmd.system.reconnect",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.factory_reset":
			val := model.ButtonActionResponse{
				Operation:       "cmd.app.factory_reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.network.get_all_nodes":
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.get_inclusion_report":
			address, err := newMsg.Payload.GetStringValue()
			if err != nil {
				// handle err
				log.Error(fmt.Errorf("Can't get strValue, error: ", err))
			}
			// for _, device := range fc.configs.DeviceCollection {
			for _, home := range fc.states.HomesAndRooms.Users[0].Homes {
				for _, room := range home.Rooms {
					for _, device := range room.Devices {
						if strconv.Itoa(device.ID) == address {
							inclReport := ns.MakeInclusionReport(strconv.Itoa(device.ID), device.Name)

							msg := fimpgo.NewMessage("evt.thing.inclusion_report", "adax", fimpgo.VTypeObject, inclReport, nil, nil, nil)
							adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
							fc.mqt.Publish(&adr, msg)
						}
					}
				}
			}
		case "cmd.thing.inclusion":
			//flag , _ := newMsg.Payload.GetBoolValue()
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.delete":
			// remove device from network
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			deviceID, ok := val["address"]
			if ok {
				// TODO: This is an example . Add your logic here or remove
				val := map[string]interface{}{
					"address": deviceID,
				}
				adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
				msg := fimpgo.NewMessage("evt.thing.exclusion_report", "adax", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
				fc.mqt.Publish(adr, msg)
				log.Info("Device with deviceID: ", deviceID, " has been removed from network.")
			} else {
				log.Error("Incorrect address")
			}

		case "cmd.app.uninstall":
			for _, home := range fc.states.HomesAndRooms.Users[0].Homes {
				for _, room := range home.Rooms {
					for _, device := range room.Devices {
						log.Info("Excluding device: ", device.ID)
						exclVal := map[string]interface{}{
							"address": strconv.Itoa(device.ID),
						}
						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "adax", ResourceAddress: "1"}
						msg := fimpgo.NewMessage("evt.thing.exclusion_report", "adax", fimpgo.VTypeObject, exclVal, nil, nil, newMsg.Payload)
						fc.mqt.Publish(adr, msg)
					}
				}
			}
		}
	}
}
