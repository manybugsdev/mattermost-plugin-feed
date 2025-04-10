package main

import (
	"os"
	"path"

	"github.com/mattermost/mattermost/server/public/model"
)

const BotName = "Feed"
const BotDisplayName = "Feed"
const BotDescription = "Bot for Feed Plugin"

func (p *Plugin) EnsureFeedBot() (string, error) {
	botId, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    BotName,
		DisplayName: BotDisplayName,
		Description: BotDescription,
	})
	p.SetFeedBotProfileImage()
	return botId, err
}

func (p *Plugin) SetFeedBotProfileImage() {
	file, err := os.Open(path.Join("assets", "icon1024x1024.png"))
	if err != nil {
		p.client.Log.Error("Error opening profile image: " + err.Error())
		return
	}
	defer file.Close()
	err = p.client.User.SetProfileImage(p.botID, file)
	if err != nil {
		p.client.Log.Error("Error setting profile image: " + err.Error())
	}
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
