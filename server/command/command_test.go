package command

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type env struct {
	client *pluginapi.Client
	api    *plugintest.API
}

func setupTest() *env {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	return &env{
		client: client,
		api:    api,
	}
}

func registerMathCommand(env *env) {
	env.api.On("RegisterCommand", &model.Command{
		Trigger:          mathCommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Evaluate a math expression",
		AutoCompleteHint: "<expression>",
		AutocompleteData: model.NewAutocompleteData(mathCommandTrigger, "<expression>", "Expression to evaluate, e.g. 2*(3+4)"),
	}).Return(nil)
	env.api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
}

const testBotUserID = "bot-user-id"

func TestMathCommandRegistration(t *testing.T) {
	assert := assert.New(t)
	env := setupTest()

	registerMathCommand(env)
	cmdHandler := NewCommandHandler(env.client, testBotUserID)
	assert.NotNil(cmdHandler)
	env.api.AssertExpectations(t)
}

func TestMathCommandSuccess(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		wantText   string
		wantInChan bool
	}{
		{name: "basic multiplication", command: "/math 2*3*4*5", wantText: "`2*3*4*5` = `120`", wantInChan: true},
		{name: "implicit x", command: "/math 2 x 2", wantText: "`2 x 2` = `4`", wantInChan: true},
		{name: "percent", command: "/math 50%", wantText: "`50%` = `0.5`", wantInChan: true},
		{name: "sin degrees", command: "/math sin(90)", wantText: "`sin(90)` = `1`", wantInChan: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			env := setupTest()
			registerMathCommand(env)
			cmdHandler := NewCommandHandler(env.client, testBotUserID)

			args := &model.CommandArgs{Command: tt.command, ChannelId: "channel-id", RootId: "root-id"}
			env.api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.UserId == testBotUserID &&
					post.ChannelId == args.ChannelId &&
					post.RootId == args.RootId &&
					post.Message == tt.wantText
			})).Return(&model.Post{}, nil)

			response, err := cmdHandler.Handle(args)
			assert.Nil(err)
			assert.Equal(&model.CommandResponse{}, response)
			env.api.AssertExpectations(t)
		})
	}
}

func TestMathCommandErrors(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantText string
	}{
		{name: "empty expression", command: "/math", wantText: "Expression is empty."},
		{name: "division by zero", command: "/math 1/0", wantText: "Result is not a finite number (e.g. division by zero)."},
		{name: "incomplete expression", command: "/math 2 +", wantText: "Could not parse expression."},
		{name: "unsupported syntax", command: "/math [1,2,3]", wantText: "Unsupported characters or syntax in expression."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			env := setupTest()
			registerMathCommand(env)
			cmdHandler := NewCommandHandler(env.client, testBotUserID)

			args := &model.CommandArgs{Command: tt.command}
			response, err := cmdHandler.Handle(args)
			assert.Nil(err)
			assert.Equal(model.CommandResponseTypeEphemeral, response.ResponseType)
			assert.Equal(tt.wantText, response.Text)
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	assert := assert.New(t)
	env := setupTest()
	registerMathCommand(env)
	cmdHandler := NewCommandHandler(env.client, testBotUserID)

	args := &model.CommandArgs{Command: "/unknown foo"}
	response, err := cmdHandler.Handle(args)
	assert.Nil(err)
	assert.Equal(model.CommandResponseTypeEphemeral, response.ResponseType)
	assert.Equal("Unknown command: /unknown foo", response.Text)
}
