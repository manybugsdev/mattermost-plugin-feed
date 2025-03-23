package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Plugin struct {
	plugin.MattermostPlugin

	client *pluginapi.Client
}

func (p *Plugin) OnActivate() error {

	p.client = pluginapi.NewClient(p.API, p.Driver)

	err := p.client.SlashCommand.Register(&model.Command{
		Trigger: "feed",
	})

	return err
}

func (p *Plugin) OnDeactivate() error {

	p.client = pluginapi.NewClient(p.API, p.Driver)

	err := p.client.SlashCommand.Register(&model.Command{
		Trigger: "feed",
	})

	p.client.SlashCommand.Unregister("", "feed")
	p.client.SlashCommand.Unregister("", "hello")

	return err
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	return &model.CommandResponse{
		Text: "Feed!",
	}, nil
}

func main() {
	plugin.ClientMain(&Plugin{})
}
