package goutube

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
)

const (
	youtubeWatchBaseURL  = "https://www.youtube.com/watch?v="
	youtubeExistsBaseURL = "https://www.youtube.com/oembed?url=http://www.youtube.com/watch?v="
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

// Video represents a YouTube video.
// To add video id and source IP address, use the AddVideoLink, AddSourceIp safe methods.
type Video struct {
	video    string
	sourceIp string
}

type streamPocketResponse struct {
	Recorded string
	Filename string
}

// AddVideoLink parse, check and add the source of a YouTube video.
// video may be a 11 long video id string, or a YouTube video link.
// Return nil if successful or ErrSource on failure.
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

// AddSourceIp parse, check and add the source IP address. Used for youtube-dl command.
// ip is the string representation of the IP address.
// Return nil if successful or ErrIp on failure.
func (v *Video) AddSourceIp(ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrIp
	}

	v.sourceIp = ip
	return nil
}

// Exists checks if the video exists using a YouTube API call.
// Return true, nil if the video exists, or false with an optional error on failure.
func (v *Video) Exists() (bool, error) {
	if v.video == "" {
		return false, nil
	}

	requestUrl := fmt.Sprintf("%s%s", youtubeExistsBaseURL, v.video)
	resp, err := http.Get(requestUrl)
	if err != nil {
		return false, err
	}

	return resp.StatusCode == 200, nil
}

// YoutubeDlLink returns the direct link of the best quality mp4 video using the youtube-dl command.
// Returns the direct link of the video and no error on success, or ("", error) on failure.
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

// StreamPocketLink returns the direct link of the best quality mp4 video using the streampocket.net API.
// Returns the direct link of the video and no error on success, or ("", error) on failure.
func (v *Video) StreamPocketLink() (string, error) {
	if v.video == "" {
		return "", ErrEmptyVideo
	}

	requestUrl := fmt.Sprintf("http://streampocket.net/json2?stream=%s%s", youtubeWatchBaseURL, v.video)

	res, err := http.Get(requestUrl)
	if err != nil {
		return "", ErrStreamPocketApi
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", ErrStreamPocketApi
	}

	err = res.Body.Close()
	if err != nil {
		return "", ErrStreamPocketApi
	}

	var response streamPocketResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return "", ErrStreamPocketResponse
	}

	return response.Recorded, nil
}

// Stream writes the content of the video on a writer using the youtube-dl command.
// wr is the writer where the video data will be written.
// Returns no error on success or an ErrEmptyVideo on failure.
func (v *Video) Stream(wr io.Writer) error {
	if v.video == "" {
		return ErrEmptyVideo
	}

	return streamBestVideo(v.video, wr)
}
