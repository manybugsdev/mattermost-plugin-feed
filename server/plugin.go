package main

import (
	"fmt"
	"io"
	"net/http"
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
const botName = "feedbot"
const botDisplayName = "Feed Bot"
const botDescription = "Feed Bot"
const fetchInterval = 20 * time.Minute

type Feed struct {
	URL       string
	Updated   int64
	ChannelID string
}

type Plugin struct {
	plugin.MattermostPlugin
	client        *pluginapi.Client
	botID         string
	backgroundJob *cluster.Job
}

func response(text string) *model.CommandResponse {
	return &model.CommandResponse{
		Text: text,
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

func (p *Plugin) saveFeeds(feeds []Feed) (bool, error) {
	return p.client.KV.Set(kvkey, feeds)
}

func (p *Plugin) loadFeeds() []Feed {
	feeds := []Feed{}
	err := p.client.KV.Get(kvkey, &feeds)
	if err != nil {
		p.client.Log.Error("Error loading feeds: " + err.Error())
	}
	return feeds
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
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

func getDate(item *gofeed.Item) *time.Time {
	date := item.UpdatedParsed
	if date == nil {
		date = item.PublishedParsed
	}
	return date
}

func (p *Plugin) fetchFeeds() {
	feeds := p.loadFeeds()
	fp := gofeed.NewParser()
	for i, feed := range feeds {
		// Don't use ParseURL, it doesn't work at https://blogs.oracle.com/oracle4engineer/rss.
		// It returns a 403 error when fetching with the user agent of gofeed.
		body, err := httpGet(feed.URL)
		if err != nil {
			p.BotPost(feed.ChannelID, fmt.Sprintf("Error fetching: %s", feed.URL))
			continue
		}
		page, err := fp.ParseString(string(body))
		if err != nil {
			p.BotPost(feed.ChannelID, fmt.Sprintf("Error parsing: %s", feed.URL))
			continue
		}
		items := page.Items
		// filter dates exists and newer than feed.Updated
		itemsValid := []*gofeed.Item{}
		for _, item := range items {
			date := getDate(item)
			if date == nil || date.Unix() <= feed.Updated {
				continue
			}
			itemsValid = append(itemsValid, item)
		}
		items = itemsValid
		for _, item := range items {
			p.BotPost(feed.ChannelID, fmt.Sprintf("%s\n%s", item.Title, item.Link))
		}
		latest := feed.Updated
		for _, item := range items {
			u := getDate(item).Unix()
			if u > latest {
				latest = u
			}
		}
		feeds[i].Updated = latest
	}
	success, _ := p.saveFeeds(feeds)
	if !success {
		p.client.Log.Error("Error saving feeds")
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

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    botName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})

	if err != nil {
		return err
	}

	p.botID = botID

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
	if p.backgroundJob != nil {
		err := p.backgroundJob.Close()
		if err != nil {
			return err
		}
	}
	err := p.client.SlashCommand.Unregister("", trigger)
	return err
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

func (p *Plugin) listFeeds(channelID string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	text := "Feeds:\n"
	for _, feed := range feeds {
		if feed.ChannelID != channelID {
			continue
		}
		text += feed.URL + "\n"
	}
	return response(text), nil
}

func (p *Plugin) addFeed(channelID string, url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	feeds = append(feeds,
		Feed{
			URL:       url,
			ChannelID: channelID,
			Updated:   time.Now().Unix(),
		})
	success, _ := p.saveFeeds(feeds)

	if success {
		return response("Feed added"), nil
	}
	return response("Feed not added"), nil
}

func (p *Plugin) delFeed(channelID string, url string) (*model.CommandResponse, *model.AppError) {
	feeds := p.loadFeeds()
	for i, feed := range feeds {
		if feed.ChannelID != channelID {
			continue
		}
		if feed.URL == url {
			feeds = slices.Delete(feeds, i, i+1)
			success, _ := p.saveFeeds(feeds)
			if success {
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
