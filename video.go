package youtubelink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type Video struct {
	video    string
	sourceIp string
}

type StreamPocketResponse struct {
	Recorded string
	Filename string
}

func (v *Video) AddVideoLink(video string) error {
	if youtubeId.MatchString(video) {
		v.video = video
		return nil
	}

	matches := youtubeLink.FindStringSubmatch(video)
	if matches != nil {
		v.video = matches[5]
		return nil
	}

	return ErrSource
}

func (v *Video) AddSourceIp(ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrIp
	}

	v.sourceIp = ip
	return nil
}

func (v *Video) YoutubeDlLink() (string, error) {
	if v.video == "" {
		return "", ErrEmptyVideo
	}

	if v.sourceIp == "" {
		return bestVideoLink(v.video)
	} else {
		return bestVideoLinkWithIp(v.video, v.sourceIp)
	}
}

func (v *Video) StreamPocketLink() (string, error) {
	if v.video == "" {
		return "", ErrEmptyVideo
	}

	requestUrl := fmt.Sprintf("http://streampocket.net/json2?stream=%s%s", youtubeBaseURL, v.video)

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

func (v *Video) Stream(wr io.Writer) error {
	if v.video == "" {
		return ErrEmptyVideo
	}

	return streamBestVideo(v.video, wr)
}
