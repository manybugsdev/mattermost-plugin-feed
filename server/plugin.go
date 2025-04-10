package main

import (
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	err := p.RegisterFeedCommand()

	if err != nil {
		return err
	}

	botID, err := p.EnsureFeedBot()

	if err != nil {
		return err
	}

	p.botID = botID

	err = p.SetFeedBotProfileImage()

	if err != nil {
		return err
	}

	job, err := p.ScheduleJob()

	if err != nil {
		return err
	}

	p.backgroundJob = job

	return err
}

func (p *Plugin) OnDeactivate() error {
	err := p.UnscheduleJob()
	err2 := p.UnregisterFeedCommand()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func main() {
	plugin.ClientMain(&Plugin{})
}
