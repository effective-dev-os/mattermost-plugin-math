package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExtractMathMention(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		wantExpr  string
		wantMatch bool
	}{
		{name: "leading mention", message: "@math 2+2", wantExpr: "2+2", wantMatch: true},
		{name: "trailing mention", message: "2+2 @math", wantExpr: "2+2", wantMatch: true},
		{name: "case-insensitive leading mention", message: "@Math 2+2", wantExpr: "2+2", wantMatch: true},
		{name: "bare mention, no expression", message: "@math", wantExpr: "", wantMatch: true},
		{name: "bare mention with surrounding whitespace", message: "  @math  ", wantExpr: "", wantMatch: true},
		{name: "mid-message mention is a no-op", message: "what is @math 2+2", wantExpr: "", wantMatch: false},
		{name: "mention as substring of longer username, leading", message: "@math2 2+2", wantExpr: "", wantMatch: false},
		{name: "mention as substring of longer username, trailing", message: "2+2 @math2", wantExpr: "", wantMatch: false},
		{name: "no mention at all", message: "2+2", wantExpr: "", wantMatch: false},
		{name: "empty message", message: "", wantExpr: "", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, ok := extractMathMention(tt.message, "math")
			assert.Equal(t, tt.wantMatch, ok)
			if ok {
				assert.Equal(t, tt.wantExpr, expr)
			}
		})
	}
}

const testHookBotUserID = "bot-user-id"

func setupHookTest() (*plugintest.API, *Plugin) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	return api, &Plugin{client: client, botUserID: testHookBotUserID}
}

func TestMessageHasBeenPosted_NoMention(t *testing.T) {
	api, p := setupHookTest()

	post := &model.Post{Id: "post-id", UserId: "user-id", ChannelId: "channel-id", Message: "2+2"}
	p.MessageHasBeenPosted(nil, post)

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestMessageHasBeenPosted_SkipsBotOwnPost(t *testing.T) {
	api, p := setupHookTest()

	post := &model.Post{Id: "post-id", UserId: testHookBotUserID, ChannelId: "channel-id", Message: "@math 2+2"}
	p.MessageHasBeenPosted(nil, post)

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestMessageHasBeenPosted_SkipsFromPluginProp(t *testing.T) {
	api, p := setupHookTest()

	post := &model.Post{
		Id:        "post-id",
		UserId:    "some-other-user",
		ChannelId: "channel-id",
		Message:   "@math 2+2",
		Props:     model.StringInterface{model.PostPropsFromPlugin: "true"},
	}
	p.MessageHasBeenPosted(nil, post)

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

func TestMessageHasBeenPosted_BareMention(t *testing.T) {
	api, p := setupHookTest()
	api.On("GetUser", "user-id").Return(&model.User{Id: "user-id", IsBot: false}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.UserId == testHookBotUserID &&
			post.ChannelId == "channel-id" &&
			post.RootId == "" &&
			post.Message == "Mention me with a math expression, e.g. `@math 2 + 2`."
	})).Return(&model.Post{}, nil)

	post := &model.Post{Id: "post-id", UserId: "user-id", ChannelId: "channel-id", Message: "@math"}
	p.MessageHasBeenPosted(nil, post)

	api.AssertExpectations(t)
}

func TestMessageHasBeenPosted_Success(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		postId   string
		rootId   string
		wantRoot string
		wantText string
	}{
		{name: "leading mention, top-level message replies in channel", message: "@math 2*3", postId: "post-id", rootId: "", wantRoot: "", wantText: "`2*3` = `6`"},
		{name: "trailing mention, existing thread reply stays in thread", message: "50% + 10 @math", postId: "post-id", rootId: "root-id", wantRoot: "root-id", wantText: "`50% + 10` = `10.5`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, p := setupHookTest()
			api.On("GetUser", "user-id").Return(&model.User{Id: "user-id", IsBot: false}, nil)
			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.UserId == testHookBotUserID &&
					post.ChannelId == "channel-id" &&
					post.RootId == tt.wantRoot &&
					post.Message == tt.wantText
			})).Return(&model.Post{}, nil)

			post := &model.Post{Id: tt.postId, RootId: tt.rootId, UserId: "user-id", ChannelId: "channel-id", Message: tt.message}
			p.MessageHasBeenPosted(nil, post)

			api.AssertExpectations(t)
		})
	}
}

func TestMessageHasBeenPosted_EvaluationError(t *testing.T) {
	api, p := setupHookTest()
	api.On("GetUser", "user-id").Return(&model.User{Id: "user-id", IsBot: false}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.UserId == testHookBotUserID &&
			post.RootId == "" &&
			post.Message == "Could not parse expression."
	})).Return(&model.Post{}, nil)

	post := &model.Post{Id: "post-id", UserId: "user-id", ChannelId: "channel-id", Message: "@math 2 +"}
	p.MessageHasBeenPosted(nil, post)

	api.AssertExpectations(t)
}
