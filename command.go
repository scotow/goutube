package youtubelink

import (
	"bytes"
	"os/exec"
)

func commandExists(name string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v " + name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func runCommand(name string, arg ...string) (string, string, error) {
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

