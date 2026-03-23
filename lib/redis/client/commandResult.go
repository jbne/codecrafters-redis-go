package redisclientlib

import resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"

type (
	commandResult struct {
		resptypes.BaseInterface
		error
	}

	CommandResult interface {
		Val() resptypes.BaseInterface
		Err() error
	}
)

func newCommandResult(val resptypes.BaseInterface, err error) CommandResult {
	return &commandResult{
		BaseInterface: val,
		error:         err,
	}
}

func (result commandResult) Val() resptypes.BaseInterface {
	return result.BaseInterface
}

func (result commandResult) Err() error {
	return result.error
}
