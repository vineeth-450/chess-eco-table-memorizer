package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/gorilla/mux"
)

// Constants used by the application
const (
	ChessECOHelpURL   = "https://www.chessgames.com/chessecohelp.html"
	CacheTTLInSeconds = 180
	CacheKey          = "MoveData"
)

// Struct to capture each move info
type moveInfo struct {
	MoveName string `json:"moveName"`
	Moves    string `json:"moves"`
}

// In memory cache object
var cache ttlcache.SimpleCache = ttlcache.NewCache()

func main() {

	log.Println("Starting Chess ECO Table Memorizer")
	port := os.Getenv("PORT")

	r := mux.NewRouter()

	r.HandleFunc("/", listAllData).Methods("GET")
	r.HandleFunc("/{code:.{1,3}\\/?}", getMoveForCode).Methods("GET")
	r.HandleFunc("/{code:.{4,}\\/?}", getNextMove).Methods("GET")

	log.Fatal(http.ListenAndServe(":"+port, r))
}

// listAllData gets data from url and writes the same to response writer
func listAllData(w http.ResponseWriter, r *http.Request) {
	response, err := http.Get(ChessECOHelpURL)
	if err != nil {
		http.Error(w, "ISE", 502)
	}

	defer response.Body.Close()

	var respBytes []byte
	respBytes, err = ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "ISE", 502)
	}

	w.Write(respBytes)
}

// getMoveForCode gets move data for the code and returns the same
func getMoveForCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	code = strings.Trim(code, "/")

	moveMap, err, statusCode := getMoveMap()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	move, ok := moveMap[code]
	if !ok {
		http.Error(w, fmt.Sprintf("Move Code '%s' Not Found. Please Enter a valid code", code), 404)
		return
	}

	w.Write([]byte(fmt.Sprintf("<b>%s</b><br>%s", move.MoveName, move.Moves)))
}

// getNextMove gets next move given code and series of moves
func getNextMove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	code = strings.Trim(code, "/")
	params := strings.Split(code, "/")
	code = params[0]
	moves := strings.Join(params[1:], " ")

	moveMap, err, statusCode := getMoveMap()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	move, ok := moveMap[code]
	if !ok {
		http.Error(w, fmt.Sprintf("Move Code '%s' Not Found. Please Enter a valid code", code), 404)
		return
	}

	if strings.HasPrefix(move.Moves, moves) {
		nextMoves := strings.Trim(move.Moves, moves)
		if len(nextMoves) > 0 {
			nextMove := strings.Split(strings.Trim(nextMoves, " "), " ")[0]
			w.Write([]byte(nextMove))
		} else {
			w.Write([]byte("No next moves available for the code " + code))
		}
	} else {
		http.Error(w, fmt.Sprintf("Invalid moves provided %s for code %s", moves, code), 404)
	}
}

// getMoveMap gets move data from Cache if present else hits the url to get the same
func getMoveMap() (map[string]moveInfo, error, int) {
	moveMap := make(map[string]moveInfo)

	val, err := cache.Get(CacheKey)
	if err != ttlcache.ErrNotFound {
		log.Println("Using Cached Data with key", CacheKey)
		err := json.Unmarshal(val.([]byte), &moveMap)
		if err != nil {
			return nil, fmt.Errorf("ISE"), 502
		}
	} else {
		log.Println("Data not found in cache, getting data from url")
		response, err := http.Get(ChessECOHelpURL)
		if err != nil {
			return nil, fmt.Errorf("ISE"), 502
		}

		defer response.Body.Close()

		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			return nil, fmt.Errorf("ISE"), 502
		}

		cache.SetTTL(time.Duration(CacheTTLInSeconds * time.Second))
		doc.Find("tr").Each(func(index int, element *goquery.Selection) {
			moveCode := element.Find("td").First().Text()
			moveInfoStrs := strings.Split(element.Find("td").Last().Text(), "\n")
			moveMap[moveCode] = moveInfo{moveInfoStrs[0], moveInfoStrs[1]}
		})

		jsonData, _ := json.Marshal(moveMap)
		log.Println("Caching Data with Key", CacheKey)
		cache.Set(CacheKey, jsonData)
	}

	return moveMap, nil, 0
}
