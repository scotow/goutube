package youtubelink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
)

const (
	youtubeBaseURL = "https://www.youtube.com/watch?v="
)

var (
	youtubeId   = regexp.MustCompile(`^[\w\-]{11}$`)
	youtubeLink = regexp.MustCompile(`^((?:https?:)?//)?((?:www|m)\.)?((?:youtube\.com|youtu.be))(/(?:[\w\-]+\?v=|embed/|v/)?)([\w\-]+)(\S+)?$`)
)

var (
	ErrSource               = errors.New("invalid YouTube video link or id")
	ErrIp                   = errors.New("invalid source ip")
	ErrEmptyVideo           = errors.New("no video specified")
	ErrStreamPocketApi      = errors.New("cannot reach remote api")
	ErrStreamPocketResponse = errors.New("invalid remote api response")
)

type Request struct {
	video    string
	sourceIp string
}

type StreamPocketResponse struct {
	Recorded string
	Filename string
}

func (r *Request) AddVideoLink(video string) error {
	if youtubeId.MatchString(video) {
		r.video = video
		return nil
	}

	matches := youtubeLink.FindStringSubmatch(video)
	if matches != nil {
		r.video = matches[5]
		return nil
	}

	return ErrSource
}

func (r *Request) AddSourceIp(ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrIp
	}

	r.sourceIp = ip
	return nil
}

func (r *Request) YoutubeDlLink() (string, error) {
	if r.video == "" {
		return "", ErrEmptyVideo
	}

	args := []string{"-f", "best", "-g"}

	if r.sourceIp != "" {
		args = append(args, "--source-address", r.sourceIp)
	}

	args = append(args, r.video)

	videoLink, stderr, err := runCommand("youtube-dl", args...)
	if err != nil {
		return stderr, err
	}

	return videoLink, nil
}

func (r *Request) StreamPocketLink() (string, error) {
	if r.video == "" {
		return "", ErrEmptyVideo
	}

	requestUrl := fmt.Sprintf("http://streampocket.net/json2?stream=%s%s", youtubeBaseURL, r.video)

	res, err := http.Get(requestUrl)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", ErrStreamPocketApi
	}

	err = res.Body.Close()
	if err != nil {
		return "", ErrStreamPocketApi
	}

	var response StreamPocketResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", ErrStreamPocketResponse
	}

	return response.Recorded, nil
}
