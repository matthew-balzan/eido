package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func BetCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	fmt.Println(optionMap, optionMap["title"].StringValue())
}
