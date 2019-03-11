package goutube

import (
	"bytes"
	"io"
	"os/exec"
)

var (
	youtubeDlCommand = "youtube-dl"
)

func SetYoutubeDlCommand(path string) {
	youtubeDlCommand = path
}

func commandExists(name string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func IsAvailable() bool {
	return commandExists(youtubeDlCommand)
}

func runCommand(out, err io.Writer, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	cmd.Stdout = out
	cmd.Stderr = err

	return cmd.Run()
}

func runCommandString(name string, arg ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	err := runCommand(&stdout, &stderr, name, arg...)
	return stdout.String(), stderr.String(), err
}

func bestVideoDefaultArgs() []string {
	return []string{"-q", "-f", "best[ext=mp4]"}
}

func bestVideoLinkDefaultArgs() []string {
	return append(bestVideoDefaultArgs(), "-g")
}

func bestVideoLink(video string) (string, error) {
	args := bestVideoLinkDefaultArgs()
	args = append(args, video)

	videoLink, stderr, err := runCommandString(youtubeDlCommand, args...)
	if err != nil {
		return stderr, err
	}

	return videoLink, nil
}

func bestVideoLinkWithIp(video string, ip string) (string, error) {
	args := bestVideoLinkDefaultArgs()
	args = append(args, "--source-address", ip, video)

	videoLink, stderr, err := runCommandString(youtubeDlCommand, args...)
	if err != nil {
		return stderr, err
	}

	return videoLink, nil
}

func streamBestVideo(video string, wr io.Writer) error {
	args := bestVideoDefaultArgs()
	args = append(args, "-o", "-", video)

	return runCommand(wr, nil, youtubeDlCommand, args...)
}
