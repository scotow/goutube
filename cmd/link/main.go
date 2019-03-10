package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/scotow/youtubelink"
	"log"
	"os"
	"strings"
)

var (
	youtubeDlFlag = flag.Bool("y", false, "use youtube-dl package")
)

func getVideos() ([]string, error) {
	args := flag.Args()
	if len(args) > 0 {
		return args, nil
	}

	videos := make([]string, 0)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		videos = append(videos, scanner.Text())
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	return videos, nil
}

func fetchLink(video string) (string, error) {
	yt := youtubelink.Video{}

	err := yt.AddVideoLink(video)
	if err != nil {
		return "", err
	}

	if *youtubeDlFlag {
		return yt.YoutubeDlLink()
	} else {
		return yt.StreamPocketLink()
	}
}

func main() {
	flag.Parse()

	if *youtubeDlFlag && !youtubelink.IsAvailable() {
		log.Fatalln("youtube-dl package is not installed or cannot be found")
	}

	videos, err := getVideos()
	if err != nil {
		log.Fatalln("cannot get video(s)", err)
	}

	hadError := false
	for _, video := range videos {
		link, err := fetchLink(video)
		if err != nil {
			hadError = true
			log.Println(err)
		} else {
			fmt.Println(strings.TrimSpace(link))
		}
	}

	if hadError {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
