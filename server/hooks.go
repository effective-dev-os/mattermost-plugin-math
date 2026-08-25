package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/effective-dev-os/mattermost-plugin-math/server/mathexpr"
)

// mentionUsernameChars is Mattermost's username charset, used to tell a genuine
// @botUsername mention apart from a longer username that merely starts or ends
// with it (e.g. "@math2" must not match).
const mentionUsernameChars = "abcdefghijklmnopqrstuvwxyz0123456789._-"

// MessageHasBeenPosted reacts to @math mentions in ordinary channel messages,
// evaluates the expression via the mathexpr package, and replies from the bot
// account — in the channel for a top-level mention, or in the thread if the
// mention was itself posted inside one.
func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	expression, ok := extractMathMention(post.Message, botUsername)
	if !ok {
		return
	}

	if post.GetProp(model.PostPropsFromPlugin) == "true" {
		return
	}

	shouldProcess, err := p.client.Post.ShouldProcessMessage(post, pluginapi.BotID(p.botUserID))
	if err != nil {
		p.client.Log.Warn("Failed to determine whether to process post", "error", err)
		return
	}
	if !shouldProcess {
		return
	}

	// Reply in the same thread if the mention was posted in one, otherwise post directly
	// in the channel (no new thread) — matches where the triggering message itself lives.
	rootID := post.RootId

	if expression == "" {
		p.postMathReply(post.ChannelId, rootID, "Mention me with a math expression, e.g. `@math 2 + 2`.")
		return
	}

	result, err := mathexpr.Evaluate(expression)
	if err != nil {
		p.client.Log.Debug("mention math evaluation failed", "error", err)
		p.postMathReply(post.ChannelId, rootID, mathErrorMessage(err))
		return
	}

	p.postMathReply(post.ChannelId, rootID, fmt.Sprintf("`%s` = `%s`", expression, mathexpr.FormatResult(result)))
}

// postMathReply posts message from the bot account, threaded under rootID if set
// (empty rootID posts directly in the channel).
func (p *Plugin) postMathReply(channelID, rootID, message string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelID,
		RootId:    rootID,
		Message:   message,
	}
	if err := p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Error("Failed to post mention reply", "error", err)
	}
}

// extractMathMention reports whether msg addresses botUsername as its first or last
// token (case-insensitive, word-boundary aware of Mattermost's username charset). A
// mention anywhere else in the message does not match: stripping a mid-message mention
// produces silently wrong results (e.g. "sqrt @math (4)" would evaluate to 2) or
// unparseable ones (e.g. "2+2 @math 3+3"), so only the two unambiguous anchor
// positions are supported (see decisions.md D-005). On a match, it returns the
// remaining text with the mention removed, trimmed, as the expression to evaluate
// (empty if the message was just the mention).
func extractMathMention(msg, botUsername string) (string, bool) {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return "", false
	}

	if rest, ok := stripLeadingMention(trimmed, botUsername); ok {
		return strings.TrimSpace(rest), true
	}
	if rest, ok := stripTrailingMention(trimmed, botUsername); ok {
		return strings.TrimSpace(rest), true
	}
	return "", false
}

func stripLeadingMention(msg, botUsername string) (string, bool) {
	token := "@" + botUsername
	if len(msg) < len(token) || !strings.EqualFold(msg[:len(token)], token) {
		return "", false
	}
	rest := msg[len(token):]
	if r, _ := utf8.DecodeRuneInString(rest); rest != "" && isMentionChar(r) {
		return "", false
	}
	return rest, true
}

func stripTrailingMention(msg, botUsername string) (string, bool) {
	token := "@" + botUsername
	if len(msg) < len(token) || !strings.EqualFold(msg[len(msg)-len(token):], token) {
		return "", false
	}
	rest := msg[:len(msg)-len(token)]
	if r, _ := utf8.DecodeLastRuneInString(rest); rest != "" && isMentionChar(r) {
		return "", false
	}
	return rest, true
}

func isMentionChar(r rune) bool {
	return strings.ContainsRune(mentionUsernameChars, unicode.ToLower(r))
}

func mathErrorMessage(err error) string {
	switch {
	case errors.Is(err, mathexpr.ErrEmptyInput):
		return "Expression is empty."
	case errors.Is(err, mathexpr.ErrTooLong):
		return "Expression is too long (max 1024 characters)."
	case errors.Is(err, mathexpr.ErrUnsupportedSyntax):
		return "Unsupported characters or syntax in expression."
	case errors.Is(err, mathexpr.ErrCompile):
		return "Could not parse expression."
	case errors.Is(err, mathexpr.ErrRuntime):
		return "Error evaluating expression."
	case errors.Is(err, mathexpr.ErrNonFiniteResult):
		return "Result is not a finite number (e.g. division by zero)."
	default:
		return "Could not evaluate expression."
	}
}
