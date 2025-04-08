package main

import "github.com/mattermost/mattermost/server/public/model"

const BotName = "feedbot"
const BotDisplayName = "Feed Bot"
const BotDescription = "Feed Bot"

func (p *Plugin) EnsureFeedBot() (string, error) {
	return p.client.Bot.EnsureBot(&model.Bot{
		Username:    BotName,
		DisplayName: BotDisplayName,
		Description: BotDescription,
	})
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
