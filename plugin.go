package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Plugin struct {
	plugin.MattermostPlugin

	client *pluginapi.Client
}

const trigger = "feed"
const desc = "Manage your feeds"
const hint = "[list|add|del] [url]"

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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	errResponse := &model.CommandResponse{
		Text: "Invalid command",
	}
	fields := strings.Fields(args.Command)
	if len(fields) < 2 {
		return errResponse, nil
	}
	if fields[0] != "/feed" {
		return errResponse, nil
	}
	return &model.CommandResponse{
		Text: "Feed!",
	}, nil
}

func main() {
	plugin.ClientMain(&Plugin{})
}
