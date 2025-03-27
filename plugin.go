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
	"github.com/mmcdole/gofeed"
)

const trigger = "feed"
const desc = "Manage your feeds"
const hint = "[list|add|del] [url]"
const kvkey = "feeds"

type Feed struct {
	Url       string
	Updated   string
	ChannelId string
}

type Plugin struct {
	plugin.MattermostPlugin
	client        *pluginapi.Client
	backgroundJob *cluster.Job
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
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
	fp := gofeed.NewParser()
	for _, feed := range feeds {
		page, err := fp.ParseURL(feed.Url)
		if err != nil {
			p.client.Post.CreatePost(&model.Post{
				ChannelId: feed.ChannelId,
				Message:   fmt.Sprintf("Error fetching: %s", feed.Url),
			})
			continue
		}
		for _, item := range page.Items {
			p.client.Post.CreatePost(&model.Post{
				ChannelId: feed.ChannelId,
				Message:   fmt.Sprintf("%s\n%s", item.Title, item.Link),
			})
		}
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if !validCommand(fields) {
		return response("Invalid command"), nil
	}

	switch fields[1] {
	case "list":
		return p.listFeeds(args.ChannelId)
	case "add":
		return p.addFeed(args.ChannelId, fields[2])
	case "del":
		return p.delFeed(args.ChannelId, fields[2])
	}
	return response("Invalid command"), nil
}

func (p *Plugin) listFeeds(channelId string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	text := "Feeds:\n"
	for _, feed := range feeds {
		if feed.ChannelId != channelId {
			continue
		}
		text += feed.Url + "\n"
	}
	return response(text), nil
}

func (p *Plugin) addFeed(channelId string, url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	feeds = append(feeds,
		Feed{
			Url:       url,
			ChannelId: channelId,
			Updated:   time.Now().Format(time.RFC3339),
		})
	sucess, _ := p.saveFeeds(feeds)

	if sucess {
		return response("Feed added"), nil
	}
	return response("Feed not added"), nil
}

func (p *Plugin) delFeed(channelId string, url string) (*model.CommandResponse, *model.AppError) {
	if url == "all" {
		err := p.client.KV.Delete(kvkey)
		if err != nil {
			return response("All feeds not deleted"), nil
		}
		return response("All feeds deleted"), nil
	}
	feeds := p.loadFeeds()
	for i, feed := range feeds {
		if feed.ChannelId != channelId {
			continue
		}
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
