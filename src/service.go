package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	"github.com/futurehomeno/fimpgo/edgeapp"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/adax/adax-api"
	"github.com/thingsplex/adax/model"
	"github.com/thingsplex/adax/router"
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
	client.Login("115965", "WfHOntjkpyKbYCrr")
	appLifecycle.SetAppState(model.AppStateRunning, nil)
	for {
		appLifecycle.WaitForState("main", model.AppStateRunning)
		log.Info("<main>Starting update loop")
		LoadStates(configs, client, states, err, mqtt)
		ticker := time.NewTicker(time.Duration(15) * time.Second)
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

	// log.Debug("access_token: ", configs.AccessToken)
	// states.Homes, states.Rooms, err = client.GetHomesAndRooms(configs.AccessToken)
	// log.Debug("Homes: ", states.Homes)
	// log.Debug("Rooms: ", states.Rooms)
	return states
}
