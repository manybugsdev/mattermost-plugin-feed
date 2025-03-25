package main

import (
	"strings"
	"time"

	"slices"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const trigger = "feed"
const desc = "Manage your feeds"
const hint = "[list|add|del] [url]"
const kvkey = "feeds"

type Feed struct {
	Url     string
	Updated string
}

type Plugin struct {
	plugin.MattermostPlugin

	client *pluginapi.Client
}

func (p *Plugin) OnActivate() error {

	p.client = pluginapi.NewClient(p.API, p.Driver)

	err := p.client.SlashCommand.Register(&model.Command{
		Trigger:          trigger,
		AutoComplete:     true,
		AutoCompleteDesc: desc,
		AutoCompleteHint: hint,
		AutocompleteData: model.NewAutocompleteData(trigger, hint, desc),
	})

	return err
}

func (p *Plugin) OnDeactivate() error {

	err := p.client.SlashCommand.Unregister("", "feed")

	return err
}

func valid(fields []string) bool {
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if !valid(fields) {
		return response("Invalid command"), nil
	}
	switch fields[1] {
	case "list":
		return p.listFeeds()
	case "add":
		return p.addFeed(fields[2])
	case "del":
		return p.delFeed(fields[2])
	}
	return response("Invalid command"), nil
}

func (p *Plugin) listFeeds() (*model.CommandResponse, *model.AppError) {
	feeds := make([]*Feed, 0)
	p.client.KV.Get(kvkey, &feeds)
	text := "Feeds:"
	for _, feed := range feeds {
		text += feed.Url + ","
	}
	return response(text), nil
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
}

func (p *Plugin) addFeed(url string) (*model.CommandResponse, *model.AppError) {
	feeds := make([]*Feed, 0)
	p.client.KV.Get(kvkey, &feeds)
	feeds = append(feeds, &Feed{Url: url, Updated: time.Now().Format(time.RFC3339)})
	sucess, _ := p.client.KV.Set(kvkey, feeds)

	if sucess {
		return response("Feed added"), nil
	}
	return response("Feed not added"), nil
}

func (p *Plugin) delFeed(url string) (*model.CommandResponse, *model.AppError) {
	feeds := make([]*Feed, 0)
	p.client.KV.Get(kvkey, &feeds)
	for i, feed := range feeds {
		if feed.Url == url {
			feeds = slices.Delete(feeds, i, i+1)
			sucess, _ := p.client.KV.Set(kvkey, feeds)
			if sucess {
				return response("Feed deleted"), nil
			}
			return response("Feed not deleted"), nil
		}
	}
	return response("Feed not found"), nil
}

func main() {
	plugin.ClientMain(&Plugin{})
}
