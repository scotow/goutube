package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/scotow/goutube"
	"github.com/tomasen/realip"
)

const (
	maxBodySize = 512
)

var (
	errBodyTooLarge  = errors.New("request body too large")
	errReadBody      = errors.New("cannot read request body")
	errVideoNotFound = errors.New("video doesn't exists")
)

var (
	portFlag          = flag.Int("p", 8080, "HTTP listening port")
	clientIpFlag      = flag.Bool("i", false, "use real client ip while using youtube-dl redirection")
	youtubeDlPathFlag = flag.String("P", "", "path to the youtube-dl command (will look in $PATH if not specified)")
	youtubeDlFlag     = flag.Bool("y", false, "use youtube-dl package for redirection feature")
	streamKeyFlag     = flag.String("k", "", "authorization token for youtube-dl video streaming (disable if empty)")
)

type distributionHandler func(*goutube.Video, http.ResponseWriter, *http.Request)
type videoMiddleware func(http.ResponseWriter, *http.Request, distributionHandler)

func authorizationMiddleware(w http.ResponseWriter, r *http.Request, m videoMiddleware, h distributionHandler) {
	if *streamKeyFlag == "" {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	header := r.Header.Get("Authorization")

	// If the header is not set or empty, reject and ask for browser authentication.
	if header == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Check for direct match (useful for curl command).
	if header == *streamKeyFlag {
		m(w, r, h)
		return
	}

	// Should be a Basic authentication.
	if strings.HasPrefix(header, "Basic") {
		part := strings.Split(header, " ")

		// Should be two parts separated by a space.
		if len(part) != 2 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// Decode right part from base64.
		keyBytes, err := base64.StdEncoding.DecodeString(part[1])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		key := string(keyBytes)

		// Remove user part if specified.
		if strings.HasPrefix(key, ":") {
			key = key[1:]
		}

		// Check if password marches the key.
		if key == *streamKeyFlag {
			m(w, r, h)
			return
		}
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func pathMiddleware(w http.ResponseWriter, r *http.Request, h distributionHandler) {
	// Extract video id from the path.
	video, exists := mux.Vars(r)["video"]
	if !exists {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	requestMiddleware(video, w, r, h)
}

func bodyMiddleware(w http.ResponseWriter, r *http.Request, h distributionHandler) {
	// Check if body is not too big or chunked.
	if r.ContentLength > maxBodySize || r.ContentLength == -1 {
		http.Error(w, errBodyTooLarge.Error(), http.StatusNotAcceptable)
		return
	}

	// Buff the all body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, errReadBody.Error(), http.StatusInternalServerError)
		return
	}

	err = r.Body.Close()
	if err != nil {
		http.Error(w, errReadBody.Error(), http.StatusInternalServerError)
		return
	}

	// Use the body as a video id/URL.
	requestMiddleware(string(body), w, r, h)
}

func requestMiddleware(video string, w http.ResponseWriter, r *http.Request, h distributionHandler) {
	// Build video object.
	yt := goutube.Video{}
	err := yt.AddVideoLink(video)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	// Check if the video exists first.
	exists, err := yt.Exists()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, errVideoNotFound.Error(), http.StatusNotFound)
		return
	}

	h(&yt, w, r)
}

func redirectMiddleware(yt *goutube.Video, w http.ResponseWriter, r *http.Request) {
	var directLink string
	var err error

	if *youtubeDlFlag {
		// Add IP address to the request if using youtube-dl.
		if *clientIpFlag {
			err := yt.AddSourceIp(realip.FromRequest(r))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		directLink, err = yt.YoutubeDlLink()
	} else {
		directLink, err = yt.StreamPocketLink()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the video direct link.
	http.Redirect(w, r, directLink, http.StatusFound)
}

func streamMiddleware(yt *goutube.Video, w http.ResponseWriter, _ *http.Request) {
	// Set header before streaming.
	w.Header().Set("Content-Type", "video/mp4")

	// Stream the video data.
	err := yt.Stream(w)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}

func redirectVarHandler(w http.ResponseWriter, r *http.Request) {
	pathMiddleware(w, r, redirectMiddleware)
}

func redirectBodyHandler(w http.ResponseWriter, r *http.Request) {
	bodyMiddleware(w, r, redirectMiddleware)
}

func streamVarHandler(w http.ResponseWriter, r *http.Request) {
	authorizationMiddleware(w, r, pathMiddleware, streamMiddleware)
}

func streamBodyHandler(w http.ResponseWriter, r *http.Request) {
	authorizationMiddleware(w, r, bodyMiddleware, streamMiddleware)
}

func main() {
	flag.Parse()

	if *youtubeDlPathFlag != "" {
		goutube.SetYoutubeDlCommand(*youtubeDlPathFlag)
	}

	if (*youtubeDlFlag || *streamKeyFlag != "") && !goutube.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	r := mux.NewRouter()

	// Add public routes.
	r.HandleFunc("/", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/{video}", redirectVarHandler).Methods(http.MethodGet)
	r.HandleFunc("/link", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/link/{video}", redirectVarHandler).Methods(http.MethodGet)
	r.HandleFunc("/redirect", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/redirect/{video}", redirectVarHandler).Methods(http.MethodGet)

	// Add private routes if a key is specified.
	if *streamKeyFlag != "" {
		r.HandleFunc("/stream", streamBodyHandler).Methods(http.MethodPost)
		r.HandleFunc("/stream/{video}", streamVarHandler).Methods(http.MethodGet)
		r.HandleFunc("/direct", streamBodyHandler).Methods(http.MethodPost)
		r.HandleFunc("/direct/{video}", streamVarHandler).Methods(http.MethodGet)
	}

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*portFlag), r))
}
