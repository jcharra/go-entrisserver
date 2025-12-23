package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/pat"
)

var Mux *pat.Router = pat.New()
var Games map[int]*Game

type Player struct {
	Id              string `json:"player_id"`
	Penalties       []int
	LastRequestTime time.Time
	LastSnapshot    string `json:"snapshot"`
	PartIndex       int
	Alive           bool `json:"alive"`
}

type Game struct {
	Id           int      `json:"game_id"`
	Running      bool     `json:"started"`
	Width        int      `json:"width"`
	Height       int      `json:"height"`
	Capacity     int      `json:"size"`
	Duckprob     float32  `json:"duck_prob"`
	Players      []Player `json:"screen_names"`
	creationTime time.Time
	parts        []int
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

func (game *Game) getPlayer(name string) *Player {
	for idx, p := range game.Players {
		if p.Id == name {
			return &game.Players[idx]
		}
	}
	return nil
}

func newGame(width int, height int, capacity int, duckprob float32) *Game {
	id := nextid()
	g := &Game{Id: id,
		Running:      false,
		Width:        width,
		Height:       height,
		Capacity:     capacity,
		Duckprob:     duckprob,
		Players:      make([]Player, 0),
		creationTime: time.Now(),
		parts:        make([]int, 0)}
	Games[id] = g
	return g
}

func nextid() int {
	for i := 0; ; i++ {
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
	Mux.Post("/sendlines", linesHandler)
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

func createRandomParts(duckprob float32, amount int) []int {
	parts := make([]int, amount)
	for i := 0; i < amount; i++ {
		if rand.Float32() <= duckprob {
			parts[i] = 0
		} else {
			parts[i] = rand.Intn(7) + 1
		}
	}
	return parts
}

// Whenever the parts queue needs to be extended, this constant
// determines how many parts will be pre-created each time.
const NUM_PARTS_RETURNED = 100

func partsHandler(w http.ResponseWriter, req *http.Request) {
	gameId, _ := strconv.Atoi(req.URL.Query().Get("game_id"))
	game := Games[gameId]

	if game == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	playerId := req.URL.Query().Get("player_id")
	player := game.getPlayer(playerId)

	if player == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if player.PartIndex+NUM_PARTS_RETURNED > len(game.parts) {
		// Parts are running out ... let's create some new ones
		newparts := createRandomParts(game.Duckprob, NUM_PARTS_RETURNED)
		game.parts = append(game.parts, newparts...)
	}

	parts := game.parts[player.PartIndex : player.PartIndex+NUM_PARTS_RETURNED]
	player.PartIndex += NUM_PARTS_RETURNED

	writeJSON(w, parts)
}

func penaltyHandler(w http.ResponseWriter, req *http.Request) {
	gameId, _ := strconv.Atoi(req.URL.Query().Get("game_id"))

	game := Games[gameId]
	if game == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	playerId := req.URL.Query().Get("player_id")
	player := game.getPlayer(playerId)

	if player == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	snapshot := req.URL.Query().Get("game_snapshot")
	player.LastSnapshot = snapshot
	player.LastRequestTime = time.Now()

	var pen int
	if len(player.Penalties) == 0 {
		pen = 0
	} else {
		pen = player.Penalties[0]
		player.Penalties = player.Penalties[1:]
	}

	writeJSON(w, map[string]int{"penalty": pen})
}

func unregistrationHandler(w http.ResponseWriter, req *http.Request) {
	gameId, _ := strconv.Atoi(req.FormValue("game_id"))
	playerId := req.FormValue("player_id")

	fmt.Println("Unregister player", playerId, "from game", gameId)

	game := Games[gameId]

	if game == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for idx, player := range game.Players {
		if player.Id == playerId {
			// Not working as range copies slice values: players.Alive = false
			game.Players[idx].Alive = false
		}
	}
}

func statusHandler(w http.ResponseWriter, req *http.Request) {
	game_id, _ := strconv.Atoi(req.URL.Query().Get("game_id"))
	writeJSON(w, Games[game_id])
}

func linesHandler(w http.ResponseWriter, req *http.Request) {
	gameId, _ := strconv.Atoi(req.FormValue("game_id"))
	game := Games[gameId]
	if game == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	playerId := req.FormValue("player_id")
	numLines, _ := strconv.Atoi(req.FormValue("num_lines"))

	for idx, p := range game.Players {
		if p.Id != playerId {
			game.Players[idx].Penalties = append(game.Players[idx].Penalties, numLines)
		}
	}

	writeJSON(w, map[string]string{"info": "Added lines"})
}

func writeJSON(w http.ResponseWriter, s interface{}) error {
	enc := json.NewEncoder(w)
	return enc.Encode(s)
}

func cleanup() {
	for gid, game := range Games {
		if !game.Running {
			if time.Now().Sub(game.creationTime) > 120*time.Second {
				log.Println("Deleted waiting game", gid, "with timestamp", game.creationTime)
				delete(Games, gid)
			}
			continue
		}

		requestMade := false
		lastRequest := time.Time{}

		for _, player := range game.Players {
			if player.LastRequestTime.After(lastRequest) {
				lastRequest = player.LastRequestTime
				requestMade = true
			}
		}

		if requestMade && time.Now().Sub(lastRequest) > 10*time.Second {
			log.Println("Deleted game due to timeout", gid)
			delete(Games, gid)
		}
	}

}

func main() {

	var port string
	if len(os.Args) > 1 {
		port = fmt.Sprintf(":%v", os.Args[1])
	} else {
		port = ":8888"
	}

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for _ = range ticker.C {
			cleanup()
		}
	}()

	err := http.ListenAndServe(port, Mux)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}
