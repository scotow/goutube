package youtubelink

import (
	"errors"
	"log"
	"net"
	"regexp"
)

var (
	youtubeId = regexp.MustCompile(`[\w\-]{11}`)
	youtubeLink = regexp.MustCompile(`^((?:https?:)?\/\/)?((?:www|m)\.)?((?:youtube\.com|youtu.be))(\/(?:[\w\-]+\?v=|embed\/|v\/)?)([\w\-]+)(\S+)?$`)
)

type Request struct {
	video string
	sourceIp string
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

	log.Println(r.video)

	return errors.New("invalid YouTube video link or id")
}

func (r *Request) AddSourceIp(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.New("invalid source ip")
	}

	r.sourceIp = ip
	return nil
}

func (r *Request) VideoLink() (string, error) {
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