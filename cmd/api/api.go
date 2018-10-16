package main

import (
	"errors"
	"flag"
	"github.com/tomasen/realip"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/scotow/youtubelink"
)

const (
	invalidMethod	= "invalid http method"
	invalidVideo 	= "invalid youtube id or link"
	invalidBody 	= "cannot read request body"
	invalidSourceIp	= "cannot set source ip"
	directLinkError	= "cannot get video direct link"
)

var (
	useClientIp *bool
)

func parseUrl(yt *youtubelink.Request, r *http.Request) error {
	return yt.AddVideoLink(string(r.URL.Path[1:]))
}

func parseBody(yt *youtubelink.Request, r *http.Request) error {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return errors.New(invalidBody)
	}

	return yt.AddVideoLink(string(body))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, invalidMethod, http.StatusMethodNotAllowed)
		return
	}

	yt := youtubelink.Request{}
	err := parseUrl(&yt, r)

	if err != nil && r.Method == "POST" {
		err = parseBody(&yt, r)
	}

	if err != nil {
		http.Error(w, invalidVideo, http.StatusNotAcceptable)
		return
	}

	if *useClientIp {
		log.Println(realip.FromRequest(r))
		err = yt.AddSourceIp(realip.FromRequest(r))
		if err != nil {
			http.Error(w, invalidSourceIp, http.StatusInternalServerError)
			return
		}
	}

	directLink, err := yt.VideoLink()

	if err != nil {
		http.Error(w, directLinkError, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, directLink, http.StatusFound)
}

func listeningAddress() string {
	port, set := os.LookupEnv("PORT")
	if !set {
		port = "8080"
	}

	return ":" + port
}

func main() {
	if !youtubelink.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	useClientIp = flag.Bool("i", false, "use real client ip")
	flag.Parse()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(listeningAddress(), nil))
}
