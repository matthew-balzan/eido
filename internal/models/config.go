package models

type Config struct {
	DiscordToken string `mapstructure:"DISCORD_TOKEN"`
	YoutubeKey   string `mapstructure:"YOUTUBE_KEY"`
}
