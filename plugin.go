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
		Trigger:          "feed",
		AutoComplete:     true,
		AutoCompleteDesc: "Say hello to someone",
		AutoCompleteHint: "[@username]",
		AutocompleteData: model.NewAutocompleteData("feed", "[@username]", "Username to say hello to"),
	})

	return err
}

func (p *Plugin) OnDeactivate() error {

	err := p.client.SlashCommand.Unregister("", "feed")

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
