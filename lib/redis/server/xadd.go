package redisserverlib

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	xadd struct {
		redistypes.DataStore
	}
)

var (
	entryIdRegexp = regexp.MustCompile(`^((\d+-\d+)|(\d+-\*)|(\*))$`)
)

func (c xadd) moniker() string {
	return "XADD"
}

func (c xadd) getUsage() string {
	return `
usage:
	XADD key field value [field value ...]
summary:
	Appends the specified stream entry to the stream at the specified key.
	If the key does not exist, XADD will create a new key with the given stream value as a side effect of running this command.
` + "\r\n"
}

func (c xadd) execute(ctx context.Context, params commandParams) commandResult {
	if len(params) < 4 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XADD requires at least 4 arguments! %s", c.getUsage())}
	}

	if len(params)%2 == 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XADD requires an odd number of arguments! %s", c.getUsage())}
	}

	key := params[1].Val
	entryIdStr := params[2].Val

	if entryIdStr == "0-0" {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD must be greater than 0-0")}
	}

	split := strings.Split(entryIdStr, "-")
	splitLen := len(split)
	if splitLen > 2 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Invalid stream ID! Too many '-' characters.")}
	}

	streamEntryId := redistypes.AddStreamEntryId{}
	if split[0] == "*" {
		if splitLen > 1 {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Invalid stream ID! Cannot use '-' when ms is generated.")}
		}

		streamEntryId.GenMs = true
		streamEntryId.GenSeq = true
	} else {
		var err error
		streamEntryId.Ms, err = strconv.ParseUint(split[0], 10, 64)
		if err != nil {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse ms part of ID as an int64! %w", err)}
		}

		if splitLen == 2 && split[1] != "*" {
			streamEntryId.Seq, err = strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse seq part of ID as an int64! %w", err)}
			}
		} else {
			streamEntryId.GenSeq = true
		}
	}

	dsVal := c.GetOrCreate(key, redistypes.NewStream, 0)
	if dsVal.Type != redistypes.TypeStream {
		return resptypes.SimpleError{Val: fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")}
	}

	return dsVal.Stream.AddEntry(streamEntryId, params[3:])
}
