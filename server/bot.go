package main

import (
	"bytes"

	_ "embed"

	"github.com/mattermost/mattermost/server/public/model"
)

const BotName = "feed"
const BotDisplayName = "Feed"
const BotDescription = "Bot for Feed Plugin"

//go:embed icon1024x1024.png
var botImage []byte

func (p *Plugin) EnsureFeedBot() (string, error) {
	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    BotName,
		DisplayName: BotDisplayName,
		Description: BotDescription,
	})
	return botID, err
}

func (p *Plugin) SetFeedBotProfileImage() error {
	return p.client.User.SetProfileImage(p.botID, bytes.NewReader(botImage))
}

func (p *Plugin) BotPost(channelID string, text string) {
	err := p.client.Post.CreatePost(&model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		Message:   text,
	})
	if err != nil {
		p.client.Log.Error("Error posting message: " + err.Error())
	}
}
