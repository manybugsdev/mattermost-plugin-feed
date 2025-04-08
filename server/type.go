package main

import (
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
)

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
