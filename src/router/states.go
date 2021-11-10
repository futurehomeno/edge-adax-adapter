package router

import (
	"github.com/futurehomeno/edge-adax-adapter/adax-api"
	log "github.com/sirupsen/logrus"
)

func (fc *FromFimpRouter) getStates() error {
	var err error
	state := adax.State{}
	fc.states.States, err = state.GetStates(fc.configs.User, fc.configs.AccessToken)
	if err != nil {
		log.Error("Can't get state from Adax. Rejecting request, error: ", err)

		return err
	}
	return nil
}

func (fc *FromFimpRouter) setTemperature(homeID int, roomID int, newTemp float64) error {
	var err error
	state := adax.State{}
	err = state.SetTemperature(fc.configs.User, homeID, roomID, newTemp, fc.configs.AccessToken)
	if err != nil {
		log.Error("Can't set temperature. Rejecting request, error: ", err)

		return err
	}
	return nil
}
