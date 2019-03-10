package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"github.com/gorilla/mux"
	"github.com/scotow/youtubelink"
	"github.com/tomasen/realip"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	maxBodySize = 512
)

var (
	errBodyTooLarge = errors.New("request body too large")
	errReadBody     = errors.New("cannot read request body")
)

var (
	portFlag          = flag.Int("p", 8080, "HTTP listening port")
	clientIpFlag      = flag.Bool("i", false, "use real client ip while using youtube-dl redirection")
	youtubeDlPathFlag = flag.String("P", "", "path to the youtube-dl command (will look in $PATH if not specified)")
	youtubeDlFlag     = flag.Bool("y", false, "use youtube-dl package for redirection feature")
	streamKeyFlag     = flag.String("k", "", "authorization token for youtube-dl video streaming (disable if empty)")
)

type distributionHandler func(*youtubelink.Video, http.ResponseWriter, *http.Request)
type videoMiddleware func(http.ResponseWriter, *http.Request, distributionHandler)

func authorizationMiddleware(w http.ResponseWriter, r *http.Request, m videoMiddleware, h distributionHandler) {
	if *streamKeyFlag == "" {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	header := r.Header.Get("Authorization")
	if header == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if header == *streamKeyFlag {
		m(w, r, h)
		return
	}

	if strings.HasPrefix(header, "Basic") {
		part := strings.Split(header, " ")
		if len(part) != 2 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		keyBytes, err := base64.StdEncoding.DecodeString(part[1])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		key := string(keyBytes)
		if strings.HasPrefix(key, ":") {
			key = key[1:]
		}

		if key == *streamKeyFlag {
			m(w, r, h)
			return
		}
	}

	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func varMiddleware(w http.ResponseWriter, r *http.Request, h distributionHandler) {
	video, exists := mux.Vars(r)["video"]
	if !exists {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	requestMiddleware(video, w, r, h)
}

func bodyMiddleware(w http.ResponseWriter, r *http.Request, h distributionHandler) {
	if r.ContentLength > maxBodySize {
		http.Error(w, errBodyTooLarge.Error(), http.StatusNotAcceptable)
		return
	}

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

	requestMiddleware(string(body), w, r, h)
}

func requestMiddleware(video string, w http.ResponseWriter, r *http.Request, h distributionHandler) {
	yt := youtubelink.Video{}
	err := yt.AddVideoLink(video)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	h(&yt, w, r)
}

func redirectMiddleware(yt *youtubelink.Video, w http.ResponseWriter, r *http.Request) {
	if *clientIpFlag {
		err := yt.AddSourceIp(realip.FromRequest(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var directLink string
	var err error
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

func streamMiddleware(yt *youtubelink.Video, w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "video/mp4")

	err := yt.Stream(w)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}

func redirectVarHandler(w http.ResponseWriter, r *http.Request) {
	varMiddleware(w, r, redirectMiddleware)
}

func redirectBodyHandler(w http.ResponseWriter, r *http.Request) {
	bodyMiddleware(w, r, redirectMiddleware)
}

func streamVarHandler(w http.ResponseWriter, r *http.Request) {
	authorizationMiddleware(w, r, varMiddleware, streamMiddleware)
}

func streamBodyHandler(w http.ResponseWriter, r *http.Request) {
	authorizationMiddleware(w, r, bodyMiddleware, streamMiddleware)
}

func main() {
	flag.Parse()

	if *youtubeDlPathFlag != "" {
		youtubelink.SetYoutubeDlCommand(*youtubeDlPathFlag)
	}

	if (*youtubeDlFlag || *streamKeyFlag != "") && !youtubelink.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	r := mux.NewRouter()
	r.HandleFunc("/", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/{video}", redirectVarHandler).Methods(http.MethodGet)
	r.HandleFunc("/link", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/link/{video}", redirectVarHandler).Methods(http.MethodGet)
	r.HandleFunc("/redirect", redirectBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/redirect/{video}", redirectVarHandler).Methods(http.MethodGet)

	if *streamKeyFlag != "" {
		r.HandleFunc("/stream", streamBodyHandler).Methods(http.MethodPost)
		r.HandleFunc("/stream/{video}", streamVarHandler).Methods(http.MethodGet)
		r.HandleFunc("/direct", streamBodyHandler).Methods(http.MethodPost)
		r.HandleFunc("/direct/{video}", streamVarHandler).Methods(http.MethodGet)
	}

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*portFlag), r))
}
