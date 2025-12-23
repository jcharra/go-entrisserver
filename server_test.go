package main

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"net/http"
	"net/url"
	"encoding/json"
	"io/ioutil"
	"strconv"
)

var (
	server   *httptest.Server
	newGameUrl string
	registrationUrl string
)

func init() {
	server = httptest.NewServer(Mux)
	newGameUrl = fmt.Sprintf("%s/new", server.URL)
	registrationUrl = fmt.Sprintf("%s/register", server.URL)
}

func TestNewGame(t *testing.T) {
	values := url.Values{};
	values.Add("dimensions", "20x30")
	values.Add("size", "3")
	values.Add("duck_prob", "0.1")

	res, err := http.PostForm(newGameUrl, values)

	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != 201 {
		t.Errorf("Unexpected status code: %d", res.StatusCode)
	}

	defer res.Body.Close()
	jsonData, err := ioutil.ReadAll(res.Body)

	var game Game
	err = json.Unmarshal(jsonData, &game)

	if err != nil {
		t.Error("No game id found")
	}

	if game.Capacity != 3 {
		t.Errorf("Wrong game capacity: %v", game.Capacity)
	}

	if game.Width != 20 {
		t.Errorf("Wrong width: %v", game.Width)
	}

	if game.Height != 30 {
		t.Errorf("Wrong height: %v", game.Height)
	}

	if game.Duckprob != 0.1 {
		t.Errorf("Wrong duck prob: %v", game.Duckprob)
	}

	/*
	Register a new player named "peter"
	 */

	registrationValues := url.Values{}
	registrationValues.Add("game_id", strconv.Itoa(game.Id))
	registrationValues.Add("screen_name", "peter")
	res2, err := http.Get(registrationUrl + "?" + registrationValues.Encode())

	if res2.StatusCode != 200 {
		t.Errorf("Unexpected status code: %d", res.StatusCode)
	}

	defer res2.Body.Close()
	jsonData, err = ioutil.ReadAll(res2.Body)

	var pid map[string]string
	err = json.Unmarshal(jsonData, &pid)

	if pid["player_id"] != "peter" {
		t.Errorf("Unexpected id '%d'", pid["player_id"])
	}

	/*
	Register one more "peter"
	 */
	res3, err := http.Get(registrationUrl + "?" + registrationValues.Encode())

	if res3.StatusCode != 200 {
		t.Errorf("Unexpected status code: %d", res.StatusCode)
	}

	defer res3.Body.Close()
	jsonData, err = ioutil.ReadAll(res3.Body)

	var pid2 map[string]string
	err = json.Unmarshal(jsonData, &pid2)

	if pid2["player_id"] != "peter_" {
		t.Errorf("Unexpected id '%d'", pid2["player_id"])
	}
}