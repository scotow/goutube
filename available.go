package youtubelink

func IsAvailable() bool {
	return commandExists("youtube-dl")
}