package youtubelink

import (
	"bytes"
	"os/exec"
)

const (
	youtubeDlCommand = "youtube-dl"
)

func commandExists(name string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func IsAvailable() bool {
	return commandExists("youtube-dl")
}

func runCommandString(name string, arg ...string) (string, string, error) {
	cmd := exec.Command(name, arg...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}

	return string(stdout.Bytes()), string(stderr.Bytes()), nil
}

func bestVideoDefaultArgs() []string {
	return []string{"-f", "best", "-g"}
}

func bestVideoLink(video string) (string, error) {
	args := bestVideoDefaultArgs()
	args = append(args, video)

	videoLink, stderr, err := runCommandString(youtubeDlCommand, args...)
	if err != nil {
		return stderr, err
	}

	return videoLink, nil
}

func bestVideoLinkWithIp(video string, ip string) (string, error) {
	args := bestVideoDefaultArgs()
	args = append(args, "--source-address", ip, video)

	videoLink, stderr, err := runCommandString(youtubeDlCommand, args...)
	if err != nil {
		return stderr, err
	}

	return videoLink, nil
}
