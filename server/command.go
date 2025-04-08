package main

import (
	"slices"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const CommandTrigger = "feed"
const CommandDescription = "Manage your feeds"
const CommandHint = "[list|add|del] [url]"

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if !validCommand(fields) {
		return response("Invalid command"), nil
	}

	switch fields[1] {
	case "list":
		return p.ListFeeds(args.ChannelId)
	case "add":
		return p.AddFeed(args.ChannelId, fields[2])
	case "del":
		return p.DelFeed(args.ChannelId, fields[2])
	}
	return response("Invalid command"), nil
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
}

func validCommand(fields []string) bool {
	if len(fields) < 2 {
		return false
	}
	if fields[0] != "/feed" {
		return false
	}
	sub := fields[1]
	if sub != "list" && sub != "add" && sub != "del" {
		return false
	}
	if sub == "list" && len(fields) != 2 {
		return false
	}
	if sub == "add" && len(fields) != 3 {
		return false
	}
	if sub == "del" && len(fields) != 3 {
		return false
	}
	return true
}

func (p *Plugin) RegisterFeedCommand() error {
	return p.client.SlashCommand.Register(&model.Command{
		Trigger:          CommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: CommandDescription,
		AutoCompleteHint: CommandHint,
		AutocompleteData: model.NewAutocompleteData(CommandTrigger, CommandHint, CommandDescription),
	})
}

func (p *Plugin) UnregisterFeedCommand() error {
	return p.client.SlashCommand.Unregister("", CommandTrigger)
}

func (p *Plugin) ListFeeds(channelID string) (*model.CommandResponse, *model.AppError) {
	feeds := p.LoadFeeds()
	text := "Feeds:\n"
	for _, feed := range feeds {
		if feed.ChannelID != channelID {
			continue
		}
		text += feed.URL + "\n"
	}
	return response(text), nil
}

func (p *Plugin) AddFeed(channelID string, url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.LoadFeeds()
	feeds = append(feeds,
		Feed{
			URL:       url,
			ChannelID: channelID,
			Updated:   time.Now().Unix(),
		})
	success, _ := p.SaveFeeds(feeds)

	if success {
		return response("Feed added"), nil
	}
	return response("Feed not added"), nil
}

func (p *Plugin) DelFeed(channelID string, url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.LoadFeeds()
	for i, feed := range feeds {
		if feed.ChannelID != channelID {
			continue
		}
		if feed.URL == url {
			feeds = slices.Delete(feeds, i, i+1)
			success, _ := p.SaveFeeds(feeds)
			if success {
				return response("Feed deleted"), nil
			}
			return response("Feed not deleted"), nil
		}
	}
	return response("Feed not found"), nil
}
