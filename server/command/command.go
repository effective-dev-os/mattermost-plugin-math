package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/effective-dev-os/mattermost-plugin-math/server/mathexpr"
)

type Handler struct {
	client *pluginapi.Client
}

type Command interface {
	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
}

const mathCommandTrigger = "math"

// Register all your slash commands in the NewCommandHandler function.
func NewCommandHandler(client *pluginapi.Client) Command {
	err := client.SlashCommand.Register(&model.Command{
		Trigger:          mathCommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Evaluate a math expression",
		AutoCompleteHint: "<expression>",
		AutocompleteData: model.NewAutocompleteData(mathCommandTrigger, "<expression>", "Expression to evaluate, e.g. 2*(3+4)"),
	})
	if err != nil {
		client.Log.Error("Failed to register command", "error", err)
	}
	return &Handler{
		client: client,
	}
}

// ExecuteCommand hook calls this method to execute the commands that were registered in the NewCommandHandler function.
func (c *Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
	fields := strings.Fields(args.Command)
	if len(fields) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Empty command",
		}, nil
	}
	trigger := strings.TrimPrefix(fields[0], "/")
	switch trigger {
	case mathCommandTrigger:
		return c.executeMathCommand(args), nil
	default:
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Unknown command: %s", args.Command),
		}, nil
	}
}

func (c *Handler) executeMathCommand(args *model.CommandArgs) *model.CommandResponse {
	expression := strings.TrimSpace(strings.TrimPrefix(args.Command, "/"+mathCommandTrigger))

	result, err := mathexpr.Evaluate(expression)
	if err != nil {
		c.client.Log.Debug("math command evaluation failed", "expression", expression, "error", err)
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         mathErrorMessage(err),
		}
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("`%s` = `%s`", expression, mathexpr.FormatResult(result)),
	}
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
