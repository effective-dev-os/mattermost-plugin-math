package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/pkg/errors"

	"github.com/effective-dev-os/mattermost-plugin-math/server/command"
	"github.com/effective-dev-os/mattermost-plugin-math/server/store/kvstore"
)

// botIconPath is the bundle-relative path to the math bot's profile image.
const botIconPath = "assets/math-bot-icon.png"

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// kvstore is the client used to read/write KV records for this plugin.
	kvstore kvstore.KVStore

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// botUserID is the user id of the bot account used to post /math results.
	botUserID string

	// commandClient is the client used to register and execute slash commands.
	commandClient command.Command

	// router is the HTTP router for handling API requests.
	router *mux.Router

	backgroundJob *cluster.Job

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	botUserID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    "math-bot",
		DisplayName: "Math Bot",
		Description: "Posts results for the /math slash command.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure math bot account")
	}
	p.botUserID = botUserID

	if err := p.setBotIcon(); err != nil {
		p.client.Log.Warn("Failed to set math bot profile image", "error", err)
	}

	p.kvstore = kvstore.NewKVStore(p.client)

	p.commandClient = command.NewCommandHandler(p.client, p.botUserID)

	p.router = p.initRouter()

	job, err := cluster.Schedule(
		p.API,
		"BackgroundJob",
		cluster.MakeWaitForRoundedInterval(1*time.Hour),
		p.runJob,
	)
	if err != nil {
		return errors.Wrap(err, "failed to schedule background job")
	}

	p.backgroundJob = job

	return nil
}

// setBotIcon reads the math bot's profile image from the plugin bundle and applies it to the
// bot account. Failure here must not fail plugin activation.
func (p *Plugin) setBotIcon() error {
	bundlePath, err := p.client.System.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "failed to get bundle path")
	}

	iconBytes, err := os.ReadFile(filepath.Join(bundlePath, botIconPath))
	if err != nil {
		return errors.Wrap(err, "failed to read bot icon")
	}

	if err := p.client.User.SetProfileImage(p.botUserID, bytes.NewReader(iconBytes)); err != nil {
		return errors.Wrap(err, "failed to set bot profile image")
	}

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.backgroundJob != nil {
		if err := p.backgroundJob.Close(); err != nil {
			p.API.LogError("Failed to close background job", "err", err)
		}
	}
	return nil
}

// This will execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandClient.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
