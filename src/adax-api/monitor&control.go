package adax

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ()

func (s *State) GetStates(userId int, accessToken string) (*State, error) {
	url := "https://adax-api-test.azurewebsites.net/r-test-api/rest/v1/state"
	method := "POST"
	userIdStr := strconv.Itoa(userId)
	payloadString := fmt.Sprintf("%s%s%s", "{\"userIds\": [", userIdStr, ", 1],\"withTemperatures\": true,\"withPowerUsage\": true}")
	payload := strings.NewReader(payloadString)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("%s%s", "Bearer ", accessToken))

	res, err := client.Do(req)
	if ProcessHTTPResponse(res, err, s) != nil {
		return nil, err
	}
	defer res.Body.Close()

	return s, nil
}

func (hr *HomesAndRooms) GetHomesAndRooms(userId int, accessToken string) (*HomesAndRooms, error) {
	url := "https://adax-api-test.azurewebsites.net/r-test-api/rest/v1/configuration"
	method := "POST"

	userIdStr := strconv.Itoa(userId)
	payload := strings.NewReader(fmt.Sprintf("%s%s%s", "{\"userIds\": [", userIdStr, ", 1]}"))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("%s%s", "Bearer ", accessToken))

	res, err := client.Do(req)

	if err = ProcessHTTPResponse(res, err, hr); err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return hr, err
}

func (s *State) SetTemperature(userId int, homeId int, roomId int, targetTemperature float64, accessToken string) error {
	url := "https://adax-api-test.azurewebsites.net/r-test-api/rest/v1/control"
	method := "PUT"

	targetTemperature *= 100
	payload := strings.NewReader(fmt.Sprintf("%s%d%s%d%s%d%s%f%s", "{\"users\": [{\"id\": ", userId, ", \"homes\": [{\"id\": ", homeId, ", \"rooms\": [{\"id\": ", roomId, ", \"targetTemperature\": ", targetTemperature, "}]}]}]}"))
	log.Debug(payload)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Error(err)

		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("%s%s", "Bearer ", accessToken))

	res, err := client.Do(req)
	log.Debug("RESPONSE FROM SETTING TEMP: ", res)
	if res.StatusCode != 200 {
		log.Error(res.Status)

		return err
	}
	if err != nil {
		log.Error(err)

		return err
	}
	defer res.Body.Close()

	return nil
}
