package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
)

const trigger = "feed"
const desc = "Manage your feeds"
const hint = "[list|add|del] [url]"
const kvkey = "feeds"
const botName = "feedbot"
const botDisplayName = "Feed Bot"
const botDescription = "Feed Bot"

type Feed struct {
	Url       string
	Updated   string
	ChannelId string
}

type Plugin struct {
	plugin.MattermostPlugin
	client        *pluginapi.Client
	botId         string
	backgroundJob *cluster.Job
}

func (p *Plugin) saveFeeds(feeds []Feed) (bool, error) {
	return p.client.KV.Set(kvkey, feeds)
}

func (p *Plugin) loadFeeds() []Feed {
	feeds := []Feed{}
	p.client.KV.Get(kvkey, &feeds)
	return feeds
}

func (p *Plugin) fetchFeeds() {
	feeds := p.loadFeeds()
	for _, feed := range feeds {
		fmt.Println("Fetching feed: " + feed.Url)
	}
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

	if err != nil {
		return err
	}

	botId, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    botName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})

	if err != nil {
		return err
	}

	p.botId = botId

	job, err := cluster.Schedule(
		p.API,
		"BackgroundJob",
		cluster.MakeWaitForRoundedInterval(20*time.Minute),
		p.fetchFeeds,
	)

	if err != nil {
		return err
	}

	p.backgroundJob = job

	return err
}

func (p *Plugin) OnDeactivate() error {

	p.client.SlashCommand.Unregister("", "feed")

	if p.backgroundJob != nil {
		p.backgroundJob.Close()

	}

	return nil
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
	feeds := p.loadFeeds()
	text := "Feeds:\n"
	for _, feed := range feeds {
		text += feed.Url + "\n"
	}
	return response(text), nil
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
}

func (p *Plugin) addFeed(url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	feeds = append(feeds, Feed{Url: url, Updated: time.Now().Format(time.RFC3339)})
	sucess, _ := p.saveFeeds(feeds)

	if sucess {
		return response("Feed added"), nil
	}
	return response("Feed not added"), nil
}

func (p *Plugin) delFeed(url string) (*model.CommandResponse, *model.AppError) {
	if url == "all" {
		err := p.client.KV.Delete(kvkey)
		if err != nil {
			return response("All feeds not deleted"), nil
		}
		return response("All feeds deleted"), nil
	}
	feeds := p.loadFeeds()
	for i, feed := range feeds {
		if feed.Url == url {
			feeds = slices.Delete(feeds, i, i+1)
			sucess, _ := p.saveFeeds(feeds)
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
