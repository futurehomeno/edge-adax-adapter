package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/futurehomeno/edge-adax-adapter/adax-api"
	"github.com/futurehomeno/edge-adax-adapter/model"
	"github.com/futurehomeno/edge-adax-adapter/router"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	"github.com/futurehomeno/fimpgo/edgeapp"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if funk.IsEmpty(workDir) {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}

	appLifecycle := model.NewAppLifecycle()
	configs := model.NewConfigs(workDir)
	states := model.NewStates(workDir)
	err := configs.LoadFromFile()
	if err != nil {
		log.Fatal(errors.Wrap(err, "can't load config file."))
	}

	client := adax.NewClient(configs.AccessToken, configs.RefreshToken)
	// client.UpdateAuthParameters(configs.MqttServerURI)

	edgeapp.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting adax----------------")
	log.Info("Work directory : ", configs.WorkDir)
	appLifecycle.PublishEvent(model.EventConfiguring, "main", nil)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	if err != nil {
		log.Error(err)
	}
	defer mqtt.Stop()

	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model.GetDiscoveryResource())
	responder.Start()

	fimpRouter := router.NewFromFimpRouter(mqtt, appLifecycle, configs, states, client)
	fimpRouter.Start()

	// Checking internet connection
	systemCheck := edgeapp.NewSystemCheck()
	err = systemCheck.WaitForInternet(time.Second * 60)
	if err != nil {
		log.Error("<main> Internet is not available, the adapter might not work.")
	}
	if configs.User != 0 {
		appLifecycle.SetConfigState(model.ConfigStateConfigured)
		appLifecycle.SetAuthState(model.AuthStateAuthenticated)
		appLifecycle.SetConnectionState(model.ConnStateConnected)
		appLifecycle.SetAppState(model.AppStateRunning, nil)

	} else {
		appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
		appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
		appLifecycle.SetConnectionState(model.ConnStateDisconnected)
		appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
	}
	for {
		appLifecycle.WaitForState("main", model.AppStateRunning)
		log.Info("<main>Starting update loop")
		LoadStates(configs, client, states, err, mqtt)
		polltime, err := strconv.Atoi(configs.PollTimeMin)
		ticker := time.NewTicker(time.Duration(polltime) * time.Minute)
		for range ticker.C {
			if appLifecycle.AppState() != model.AppStateRunning {
				break
			}
			states = LoadStates(configs, client, states, err, mqtt)
		}
		ticker.Stop()
	}
}

func LoadStates(configs *model.Configs, client *adax.Client, states *model.States, err error, mqtt *fimpgo.MqttTransport) *model.States {
	hr := adax.HomesAndRooms{}
	s := adax.State{}
	lastStates := states.States

	states.HomesAndRooms = nil
	states.States = nil

	if configs.AccessToken == "" {
		RefreshTokens(configs, client, err)

		return states
	}

	if configs.User != 0 {
		states.HomesAndRooms, err = hr.GetHomesAndRooms(configs.User, configs.AccessToken)
		if err != nil {
			RefreshTokens(configs, client, err)
		}
		states.States, err = s.GetStates(configs.User, configs.AccessToken)
		if err != nil {
			log.Error("error: ", err)
		}

		for i, home := range states.States.Users[0].Homes {
			for p, room := range home.Rooms {
				for k, device := range room.Devices {
					currentTemp := float32(room.Temperature) / 100
					id := strconv.Itoa(device.ID)
					props := fimpgo.Props{}
					props["unit"] = "C"

					if lastStates != nil {
						lastTemp := float32(lastStates.Users[0].Homes[i].Rooms[p].Temperature) / 100
						if lastTemp != currentTemp {
							adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_temp", ServiceAddress: id}
							msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", fimpgo.VTypeFloat, currentTemp, props, nil, nil)
							if err := mqtt.Publish(adr, msg); err != nil {
								log.Error(err)
							}

							log.Debug("last temp: ", lastTemp)
							log.Debug("current temp: ", currentTemp)
							log.Debug("New temp. evt.sensor.report sent")
						}
					} else {
						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_temp", ServiceAddress: id}
						msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", fimpgo.VTypeFloat, currentTemp, props, nil, nil)
						if err := mqtt.Publish(adr, msg); err != nil {
							log.Error(err)
						}

						log.Info("Initial sensor report sent")
					}

					setpointTemp := fmt.Sprintf("%f", (float32(room.TargetTemperature) / 100))
					setpointVal := map[string]interface{}{
						"type": "heat",
						"temp": setpointTemp,
						"unit": "C",
					}
					if setpointTemp != "0" {
						if lastStates != nil {
							lastSetpoint := fmt.Sprintf("%f", float32(lastStates.Users[0].Homes[i].Rooms[p].TargetTemperature)/100)
							if lastSetpoint != setpointTemp {
								adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: id}
								msg := fimpgo.NewMessage("evt.setpoint.report", "thermostat", fimpgo.VTypeStrMap, setpointVal, nil, nil, nil)
								if err := mqtt.Publish(adr, msg); err != nil {
									log.Error(err)
								}

								log.Debug("last setpoint: ", lastSetpoint)
								log.Debug("current setpoint: ", setpointTemp)
								log.Info("New setpoint. evt.setpoint.report sent")
							}
						} else {
							adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "thermostat", ServiceAddress: id}
							msg := fimpgo.NewMessage("evt.setpoint.report", "thermostat", fimpgo.VTypeStrMap, setpointVal, nil, nil, nil)
							if err := mqtt.Publish(adr, msg); err != nil {
								log.Error(err)
							}
							log.Info("Initial setpoint report sent")
						}
					}

					currentEnergy := float32(device.PowerUsage.Energy) / 1000
					props = fimpgo.Props{}
					props["unit"] = "kWh"

					if lastStates != nil {
						lastEnergy := float32(lastStates.Users[0].Homes[i].Rooms[p].Devices[k].PowerUsage.Energy) / 1000
						if lastEnergy != currentEnergy {
							adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "meter_elec", ServiceAddress: id}
							msg := fimpgo.NewMessage("evt.meter.report", "meter_elec", fimpgo.VTypeFloat, currentEnergy, props, nil, nil)
							if err := mqtt.Publish(adr, msg); err != nil {
								log.Error(err)
							}
							log.Debug("last energy: ", lastEnergy)
							log.Debug("current energy: ", currentEnergy)
							log.Debug("New energy. evt.meter.report sent")
						}
					} else {
						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "meter_elec", ServiceAddress: id}
						msg := fimpgo.NewMessage("evt.meter.report", "meter_elec", fimpgo.VTypeFloat, currentEnergy, props, nil, nil)
						if err := mqtt.Publish(adr, msg); err != nil {
							log.Error(err)
						}
						log.Debug("Initial meter report sent")
					}
				}
			}
		}
	}
	if err = configs.SaveToFile(); err != nil {
		log.Error("Can't save to config file")
	}
	log.Debug("")

	return states
}

func RefreshTokens(configs *model.Configs, client *adax.Client, err error) {
	log.Error("Deleting token and trying to get new")
	configs.AccessToken = ""
	configs.RefreshToken = ""
	configs.Code = ""

	configs.Code, err = client.GetCode()
	if err != nil {
		log.Error(err)
		log.Error("Can't get new code")
		return
	}
	log.Debug("NEW CODE: ", configs.Code)
	configs.AccessToken, configs.RefreshToken, err = client.GetTokens(configs.Code)
	if err != nil {
		log.Error(err)
		log.Error("Can't get new tokens.")

		return
	}
	log.Info("New access token: ", configs.AccessToken)
	log.Info("New refresh token: ", configs.RefreshToken)

	return
}
