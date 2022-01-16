package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
)

type moveInfo struct {
	moveName string
	moves    string
}

func main() {

	fmt.Println("Chess ECO Table Memorizer")
	r := mux.NewRouter()
	r.HandleFunc("/", listAllData).Methods("GET")
	r.HandleFunc("/{code}", getMoveForCode).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func listAllData(w http.ResponseWriter, r *http.Request) {
	response, err := http.Get("https://www.chessgames.com/chessecohelp.html")
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

	response, err := http.Get("https://www.chessgames.com/chessecohelp.html")
	if err != nil {
		http.Error(w, "ISE", 502)
	}

	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		http.Error(w, "ISE", 502)
	}

	moveMap := make(map[string]moveInfo)
	doc.Find("tr").Each(func(index int, element *goquery.Selection) {
		moveCode := element.Find("td").First().Text()
		moveInfoStrs := strings.Split(element.Find("td").Last().Text(), "\n")
		moveMap[moveCode] = moveInfo{moveInfoStrs[0], moveInfoStrs[1]}
	})

	w.Write([]byte(fmt.Sprintf("<b>%s</b><br>%s", moveMap[code].moveName, moveMap[code].moves)))
}
