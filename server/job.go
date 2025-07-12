package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mmcdole/gofeed"
)

const JobInterval = 20 * time.Minute

func (p *Plugin) ScheduleJob() (*cluster.Job, error) {
	return cluster.Schedule(
		p.API,
		"BackgroundJob",
		cluster.MakeWaitForRoundedInterval(JobInterval),
		p.FetchFeeds,
	)
}

func (p *Plugin) UnscheduleJob() error {
	if p.backgroundJob == nil {
		return nil
	}
	return p.backgroundJob.Close()
}

func getDate(item *gofeed.Item) *time.Time {
	date := item.PublishedParsed
	if date == nil {
		date = item.UpdatedParsed
	}
	return date
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

func (p *Plugin) FetchFeeds() {
	feeds := p.LoadFeeds()
	fp := gofeed.NewParser()
	for i, feed := range feeds {
		// Don't use ParseURL, it doesn't work at https://blogs.oracle.com/oracle4engineer/rss.
		// It returns a 403 error when fetching with the user agent of gofeed.
		body, err := httpGet(feed.URL)
		if err != nil {
			p.client.Log.Error(fmt.Sprintf("Error fetching: %s", feed.URL))
			continue
		}
		page, err := fp.ParseString(string(body))
		if err != nil {
			p.client.Log.Error(fmt.Sprintf("Error parsing: %s", feed.URL))
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
			p.BotPost(feed.ChannelID, fmt.Sprintf("%s | %s\n%s", item.Title, page.Title, item.Link))
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
	success, _ := p.SaveFeeds(feeds)
	if !success {
		p.client.Log.Error("Error saving feeds")
	}
}
