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

const (
	ChessECOHelpURL   = "https://www.chessgames.com/chessecohelp.html"
	CacheTTLInSeconds = 180
	CacheKey          = "MoveData"
)

type moveInfo struct {
	MoveName string `json:"moveName"`
	Moves    string `json:"moves"`
}

var cache ttlcache.SimpleCache = ttlcache.NewCache()

func main() {

	fmt.Println("Chess ECO Table Memorizer")
	port := os.Getenv("PORT")
	r := mux.NewRouter()
	r.HandleFunc("/", listAllData).Methods("GET")
	r.HandleFunc("/{code}", getMoveForCode).Methods("GET")

	log.Fatal(http.ListenAndServe(":"+port, r))
}

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

func getMoveForCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	moveMap := make(map[string]moveInfo)

	val, err := cache.Get(CacheKey)
	if err != ttlcache.ErrNotFound {
		log.Println("Using Cached Data with key", CacheKey)
		err := json.Unmarshal(val.([]byte), &moveMap)
		if err != nil {
			http.Error(w, "ISE", 502)
			return
		}
	} else {
		log.Println("Data not found in cache, getting data from url")
		response, err := http.Get(ChessECOHelpURL)
		if err != nil {
			http.Error(w, "ISE", 502)
			return
		}

		defer response.Body.Close()

		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			http.Error(w, "ISE", 502)
			return
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

	move, ok := moveMap[code]
	if !ok {
		http.Error(w, fmt.Sprintf("Move Code '%s' Not Found. Please Enter a valid code", code), 404)
		return
	}

	w.Write([]byte(fmt.Sprintf("<b>%s</b><br>%s", move.MoveName, move.Moves)))
}
