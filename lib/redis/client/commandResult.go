package redisclientlib

import resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"

type (
	commandResult struct {
		resptypes.RespSerializable
		error
	}

	CommandResult interface {
		Val() resptypes.RespSerializable
		Err() error
	}
)

func newCommandResult(val resptypes.RespSerializable, err error) CommandResult {
	return &commandResult{
		RespSerializable: val,
		error:            err,
	}
}

func (result commandResult) Val() resptypes.RespSerializable {
	return result.RespSerializable
}

func (result commandResult) Err() error {
	return result.error
}
