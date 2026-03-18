package redisserverlib

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	redistypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/redis"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	xadd struct {
		*redisDataStore
	}
)

var (
	entryIdRegexp = regexp.MustCompile(`^((?P<0>\d+-\d+)|(?P<1>\d+-\*)|(?P<2>\*))$`)
)

func (c xadd) getUsage(ctx context.Context) string {
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
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XADD requires at least 4 arguments! %s", c.getUsage(ctx))}
	}

	if len(params)%2 == 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR XADD requires an odd number of arguments! %s", c.getUsage(ctx))}
	}

	streamKey := params[1].Val
	entryId := params[2].Val

	match := entryIdRegexp.FindStringSubmatch(entryId)
	if match == nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD is not in a valid format! %s", c.getUsage(ctx))}
	}

	subMatchMap := make(map[string]string)
	for i, name := range entryIdRegexp.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}

	if subMatchMap["0"] == "" {
		nextMsPart := time.Now().UnixMilli()
		if subMatchMap["1"] != "" {
			var err error
			nextMsPart, err = strconv.ParseInt(entryId[0:len(entryId)-2], 10, 64)
			if err != nil {
				return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse milliseconds part of ID as an int64! %w", err)}
			}
		}

		nextSeqPart := int64(0)
		if nextMsPart == c.lastStreamEntryId.Ms {
			nextSeqPart = c.lastStreamEntryId.Seq + 1
		}

		entryId = fmt.Sprintf("%d-%d", nextMsPart, nextSeqPart)
	}

	if strings.Compare(entryId, "0-0") <= 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD must be greater than 0-0")}
	}

	if strings.Compare(entryId, c.lastStreamEntryId.Id) <= 0 {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")}
	}

	var msPart int64
	var seqPart int64
	var err error
	entryIdParts := strings.Split(entryId, "-")

	msPart, err = strconv.ParseInt(entryIdParts[0], 10, 64)
	if err != nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse milliseconds part of ID as an int64! %w", err)}
	}

	seqPart, err = strconv.ParseInt(entryIdParts[1], 10, 64)
	if err != nil {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR Could not parse sequence part of ID as an int64! %w", err)}
	}

	streamAny := c.dataStore.GetOrCreate(streamKey, redistypes.NewStream, 0)
	if stream, ok := streamAny.(redistypes.Stream); ok {

		if _, exists := stream.Get(entryId); exists {
			return resptypes.SimpleError{Val: fmt.Errorf("ERR Entry with ID '%s' already exists in stream! %s", entryId, c.getUsage(ctx))}
		}

		entry := stream.GetOrCreate(entryId, redistypes.NewStreamEntries, 0)
		for i := 3; i < len(params); i += 2 {
			entry.PushBack(params[i : i+2])
		}

		c.lastStreamEntryId = redistypes.StreamEntryId{Id: entryId, Ms: msPart, Seq: seqPart}
		return resptypes.NewBulkString(entryId)
	}

	return resptypes.SimpleError{Val: fmt.Errorf("ERR XADD can only be called on streams! %s", c.getUsage(ctx))}
}
