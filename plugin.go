package main

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
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
const botName = "feedbot"
const botDisplayName = "Feed Bot"
const botDescription = "Feed Bot"
const fetchInterval = 20 * time.Minute

type Feed struct {
	Url       string
	Updated   int64
	ChannelId string
}

type Plugin struct {
	plugin.MattermostPlugin
	client        *pluginapi.Client
	botId         string
	backgroundJob *cluster.Job
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
	}
}

func (p *Plugin) BotPost(channelId string, text string) {
	p.client.Post.CreatePost(&model.Post{
		UserId:    p.botId,
		ChannelId: channelId,
		Message:   text,
	})
}

func (p *Plugin) saveFeeds(feeds []Feed) (bool, error) {
	return p.client.KV.Set(kvkey, feeds)
}

func (p *Plugin) loadFeeds() []Feed {
	feeds := []Feed{}
	p.client.KV.Get(kvkey, &feeds)
	return feeds
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil

}

func (p *Plugin) fetchFeeds() {
	feeds := p.loadFeeds()
	fp := gofeed.NewParser()
	for _, feed := range feeds {
		// Don't use ParseURL, it doesn't work at https://blogs.oracle.com/oracle4engineer/rss.
		// It returns a 403 error when fetching with the user agent of gofeed.
		body, err := httpGet(feed.Url)
		if err != nil {
			p.BotPost(feed.ChannelId, fmt.Sprintf("Error fetching: %s", feed.Url))
			continue
		}
		page, err := fp.ParseString(string(body))
		if err != nil {
			p.BotPost(feed.ChannelId, fmt.Sprintf("Error parsing: %s", feed.Url))
			continue
		}
		items := page.Items
		sort.Slice(items, func(i, j int) bool {
			return items[i].UpdatedParsed.Before(*items[j].UpdatedParsed)
		})
		for _, item := range items {
			if item.UpdatedParsed == nil {
				continue
			}
			if item.UpdatedParsed.Unix() < feed.Updated {
				continue
			}
			feed.Updated = item.UpdatedParsed.Unix()
			p.BotPost(feed.ChannelId, fmt.Sprintf("New feed: %s\n%s", item.Title, item.Link))

		}
	}
	p.saveFeeds(feeds)
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
		cluster.MakeWaitForRoundedInterval(fetchInterval),
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
			Updated:   time.Now().Unix(),
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
