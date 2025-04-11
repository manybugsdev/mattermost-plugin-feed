package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const CommandTrigger = "feed"
const CommandDescription = "Manage your feeds"

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	commands := strings.Fields(args.Command)
	if len(commands) < 2 || commands[0] != "/feed" {
		return responseHelp(), nil
	}
	subCommand := commands[1]
	switch subCommand {
	case "help":
		return responseHelp(), nil
	case "list":
		return p.ListFeeds(args), nil
	}
	if len(commands) != 3 {
		return responseHelp(), nil
	}
	switch subCommand {
	case "add":
		return p.AddFeed(args, commands[2]), nil
	case "del":
		return p.DelFeed(args, commands[2]), nil
	}
	return responseHelp(), nil
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
}

func responseHelp() *model.CommandResponse {
	return response("```" + `
Usage: /feed <command> [args]
/feed list
	List all feeds
/feed add <url>
	Add a feed
/feed del <url_or_index>
	Delete a feed
/feed help
	Show this help
` + "```")
}

func (p *Plugin) GetUserName(userID string) string {
	user, err := p.client.User.Get(userID)
	if err != nil {
		return "anonymous"
	}
	return user.Username
}

func (p *Plugin) RegisterFeedCommand() error {
	return p.client.SlashCommand.Register(&model.Command{
		Trigger:          CommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: CommandDescription,
	})
}

func (p *Plugin) UnregisterFeedCommand() error {
	return p.client.SlashCommand.Unregister("", CommandTrigger)
}

func (p *Plugin) ListFeeds(args *model.CommandArgs) *model.CommandResponse {
	feeds := p.LoadFeeds()
	text := "Feeds in this channel:\n\n"
	for _, feed := range feeds {
		if feed.ChannelID != args.ChannelId {
			continue
		}
		text += "1. " + feed.URL + "\n"
	}
	return response(text)
}

func (p *Plugin) AddFeed(args *model.CommandArgs, url string) *model.CommandResponse {
	feeds := p.LoadFeeds()
	feeds = append(feeds,
		Feed{
			URL:       url,
			ChannelID: args.ChannelId,
			Updated:   time.Now().Unix(),
		})
	success, _ := p.SaveFeeds(feeds)
	if success {
		userName := p.GetUserName(args.UserId)
		p.BotPost(args.ChannelId,
			"**New feed added!**\n\n"+url+" by @"+userName)
		return response("")
	}
	return response("Error: unable to save feeds")
}

func (p *Plugin) DelFeed(args *model.CommandArgs, urlOrIndex string) *model.CommandResponse {
	feeds := p.LoadFeeds()
	for i, feed := range feeds {
		if feed.ChannelID != args.ChannelId {
			continue
		}
		if feed.URL == urlOrIndex || fmt.Sprint(i+1) == urlOrIndex {
			feeds = slices.Delete(feeds, i, i+1)
			success, _ := p.SaveFeeds(feeds)
			if success {
				userName := p.GetUserName(args.UserId)
				p.BotPost(args.ChannelId, "**Feed deleted!**\n\n"+feed.URL+" by @"+userName)
				return response("")
			}
			return response("Error: unable to save feeds")
		}
	}
	return response(urlOrIndex + " is not found in this channel. Please check the URL and try again.")
}
