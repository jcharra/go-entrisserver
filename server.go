package main

import (
	"net/http"
	"log"
	"github.com/gorilla/pat"
	"fmt"
	"strconv"
	"strings"
	"encoding/json"
	"time"
	"errors"
)

var Mux *pat.Router = pat.New()
var Games map[int]*Game

type Player struct {
	Id              string    `json:"player_id"`
	Penalties       []int
	LastRequestTime time.Time
	LastSnapshot    string    `json:"snapshot"`
	PartIndex       int
	Alive			bool
}

type Game struct {
	Id       int      `json:"game_id"`
	Running  bool     `json:"started"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Capacity int      `json:"size"`
	Duckprob float32  `json:"duck_prob"`
	Players  []Player `json:"screen_names"`
}

func (game *Game) addPlayer(name string) (string, error) {
	if len(game.Players) == game.Capacity {
		return "", errors.New("Game is full")
	}

	for i := 0; i < len(game.Players); i++ {
		if game.Players[i].Id == name {
			return game.addPlayer(name + "_")
		}
	}

	player := Player{Id: name, Penalties: make([]int, 10), Alive: true}
	game.Players = append(game.Players, player)
	return name, nil
}

func newGame(width int, height int, capacity int, duckprob float32) *Game {
	id := nextid()
	g := &Game{Id: id,
		Running: false,
		Width: width,
		Height: height,
		Capacity: capacity,
		Duckprob: duckprob,
		Players: make([]Player, 0)}
	Games[id] = g
	return g
}

func nextid() int {
	for i := 0;; i++ {
		if Games[i] == nil {
			return i
		}
	}
}

func init() {
	Mux.Post("/new", newGameHandler)
	Mux.Get("/register", registrationHandler)
	Mux.Post("/unregister", unregistrationHandler)
	Mux.Get("/list", gamesListHandler)
	Mux.Get("/getparts", partsHandler)
	Mux.Get("/receive", penaltyHandler)
	Mux.Get("/sendlines", linesHandler)
	Mux.Get("/status", statusHandler)

	Games = make(map[int]*Game)
}

func newGameHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	dim := req.FormValue("dimensions")
	parts := strings.Split(dim, "x")
	width, _ := strconv.Atoi(parts[0])
	height, _ := strconv.Atoi(parts[1])

	cap, _ := strconv.Atoi(req.FormValue("size"))
	prob, _ := strconv.ParseFloat(req.FormValue("duck_prob"), 64)
	g := newGame(width, height, cap, float32(prob))

	w.WriteHeader(http.StatusCreated)

	err := writeJSON(w, g)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func registrationHandler(w http.ResponseWriter, req *http.Request) {
	gameId, _ := strconv.Atoi(req.URL.Query().Get("game_id"))
	playerName := req.URL.Query().Get("screen_name")

	game := Games[gameId]

	if game != nil {
		player_id, err := game.addPlayer(playerName)

		if err == nil {
			fmt.Printf("Registered player %v for game %v\n", player_id, gameId)
			writeJSON(w, map[string]string{"player_id": player_id})

			// Start game as soon as all players are seated
			if len(game.Players) == game.Capacity {
				game.Running = true
			}

		} else {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotAcceptable)
		}

	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func gamesListHandler(w http.ResponseWriter, req *http.Request) {
	jsonMap := make(map[string]Game)

	for id, game := range Games {
		jsonMap[strconv.Itoa(id)] = *game
	}

	writeJSON(w, jsonMap)
}

func partsHandler(w http.ResponseWriter, req *http.Request) {
	// dummy ... :) get game ID and duck prob, then get or create parts
	// according to player's part index.
	writeJSON(w, []int{1, 2, 3})
}

func penaltyHandler(w http.ResponseWriter, req *http.Request) {
	// dummy
	writeJSON(w, map[string]int{"penalty": 0})
}

func unregistrationHandler(w http.ResponseWriter, req *http.Request) {
	game_id, _ := strconv.Atoi(req.URL.Query().Get("game_id"))
	player_id := req.URL.Query().Get("player_id")

	fmt.Println("Unregister player", player_id, " from game ", game_id)

	game := Games[game_id]
	for _, player := range game.Players {
		if player.Id == player_id {
			// TODO: Y u no work?
			player.Alive = false
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func statusHandler(w http.ResponseWriter, req *http.Request) {
	game_id, _ := strconv.Atoi(req.URL.Query().Get("game_id"))
	writeJSON(w, Games[game_id])
}

func linesHandler(w http.ResponseWriter, req *http.Request) {

}

func writeJSON(w http.ResponseWriter, s interface{}) error {
	enc := json.NewEncoder(w)
	return enc.Encode(s)
}

func main() {

	err := http.ListenAndServe(":8888", Mux)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}
