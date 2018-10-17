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

var (
	youtubeId = regexp.MustCompile(`^[\w\-]{11}$`)
	youtubeLink = regexp.MustCompile(`^((?:https?:)?\/\/)?((?:www|m)\.)?((?:youtube\.com|youtu.be))(\/(?:[\w\-]+\?v=|embed\/|v\/)?)([\w\-]+)(\S+)?$`)
)

type Request struct {
	video string
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

	return errors.New("invalid YouTube video link or id")
}

func (r *Request) AddSourceIp(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.New("invalid source ip")
	}

	r.sourceIp = ip
	return nil
}

func (r *Request) YoutubeDlLink() (string, error) {
	if r.video == "" {
		return "", errors.New("no video specified")
	}

	args := []string {"-f", "best", "-g"}

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
		return "", errors.New("no video specified")
	}

	requestUrl := fmt.Sprintf("http://streampocket.net/json2?stream=https://www.youtube.com/watch?v=%s", r.video)

	res, err := http.Get(requestUrl)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return "", errors.New("cannot reach remote api")
	}

	var response StreamPocketResponse
	err = json.Unmarshal(data, &response)

	if err != nil {
		return "", errors.New("invalid remote api response")
	}

	return response.Recorded, nil
}