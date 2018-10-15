package main

import (
	"github.com/scotow/youtubelink"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	invalidVideo 	= "invalid youtube id or link"
	invalidBody 	= "cannot read request body"
	directLinkError	= "cannot get video direct link"
)

func parseGetRequest(w http.ResponseWriter, r *http.Request) {
	request := youtubelink.Request{}
	err := request.AddVideoLink(string(r.URL.Path[1:]))

	if err != nil {
		http.Error(w, invalidVideo, http.StatusNotAcceptable)
		return
	}

	directLink, err := request.VideoLink()

	if err != nil {
		http.Error(w, directLinkError, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, directLink, http.StatusFound)
}

func parsePostRequest(w http.ResponseWriter, r *http.Request) {
	request := youtubelink.Request{}
	err := request.AddVideoLink(r.URL.Path[1:])

	if err != nil {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			http.Error(w, invalidBody, http.StatusInternalServerError)
			return
		}

		err = request.AddVideoLink(string(body))
		if err != nil {
			http.Error(w, invalidVideo, http.StatusNotAcceptable)
			return
		}
	}

	directLink, err := request.VideoLink()

	if err != nil {
		http.Error(w, directLinkError, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, directLink, http.StatusFound)
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		parseGetRequest(w, r)
	case "POST":
		parsePostRequest(w, r)
	}
}

func main() {
	if !youtubelink.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
