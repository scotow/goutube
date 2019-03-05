package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/scotow/youtubelink"
	"github.com/tomasen/realip"
)

const (
	maxBodySize = 512
)

var (
	errInvalidMethod = errors.New("invalid http method")
	errBodyTooLarge  = errors.New("request body too large")
	errReadBody      = errors.New("cannot read request body")
)

var (
	portFlag      = flag.Int("p", 8080, "listening port")
	clientIpFlag  = flag.Bool("i", false, "use real client ip")
	youtubeDlFlag = flag.Bool("y", false, "use youtube-dl package")
)

func parseUrl(yt *youtubelink.Request, r *http.Request) error {
	return yt.AddVideoLink(string(r.URL.Path[1:]))
}

func parseBody(yt *youtubelink.Request, r *http.Request) error {
	if r.ContentLength > maxBodySize {
		return errBodyTooLarge
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errReadBody
	}

	err = r.Body.Close()
	if err != nil {
		return errReadBody
	}

	return yt.AddVideoLink(string(body))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, errInvalidMethod.Error(), http.StatusMethodNotAllowed)
		return
	}

	yt := youtubelink.Request{}
	err := parseUrl(&yt, r)

	if err == youtubelink.ErrSource && r.Method == "POST" {
		err = parseBody(&yt, r)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	if *clientIpFlag {
		err = yt.AddSourceIp(realip.FromRequest(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var directLink string
	if *youtubeDlFlag {
		directLink, err = yt.YoutubeDlLink()
	} else {
		directLink, err = yt.StreamPocketLink()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, directLink, http.StatusFound)
}

func main() {
	flag.Parse()

	if *youtubeDlFlag && !youtubelink.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*portFlag), nil))
}