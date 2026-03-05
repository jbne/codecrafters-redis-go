package redisserverlib

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type (
	xadd struct{}
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

func (c xadd) execute(ctx context.Context, r *redisCommandProcessor, params commandParams) commandResult {
	if len(params) < 4 {
		return fmt.Sprintf("-ERR XADD requires at least 4 arguments! %s", c.getUsage(ctx))
	}

	if len(params)%2 == 0 {
		return fmt.Sprintf("-ERR XADD requires an odd number of arguments! %s", c.getUsage(ctx))
	}

	streamKey := params[1]
	entryId := params[2]

	match := entryIdRegexp.FindStringSubmatch(entryId)
	if match == nil {
		return fmt.Sprintf("-ERR The ID specified in XADD is not in a valid format! %s", c.getUsage(ctx))
	}

	subMatchMap := make(map[string]string)
	for i, name := range entryIdRegexp.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}

	if subMatchMap["0"] != "" {
		if strings.Compare(entryId, "0-0") <= 0 {
			return "-ERR The ID specified in XADD must be greater than 0-0\r\n"
		}
	} else if subMatchMap["1"] == "" && subMatchMap["2"] == "" {
		return fmt.Sprintf("-ERR The ID format specified was valid but is not supported! %s", c.getUsage(ctx))
	} else {
		nextMsPart := time.Now().UnixMilli()
		if subMatchMap["1"] != "" {
			var err error
			nextMsPart, err = strconv.ParseInt(entryId[0 : len(entryId)-2], 10, 64)
			if err != nil {
				return fmt.Sprintf("-ERR Could not parse milliseconds part of ID as an int! %s", err)
			}
		}

		nextSeqPart := int64(0)
		if nextMsPart == r.LastEntryId.ms {
			nextSeqPart = r.LastEntryId.seq + 1
		}

		entryId = fmt.Sprintf("%d-%d", nextMsPart, nextSeqPart)
	}

	if strings.Compare(entryId, r.LastEntryId.id) <= 0 {
		return "-ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n"
	}

	var msPart int64
	var seqPart int64
	var err error
	entryIdParts := strings.Split(entryId, "-")

	msPart, err = strconv.ParseInt(entryIdParts[0], 10, 64)
	if err != nil {
		return fmt.Sprintf("-ERR Could not parse milliseconds part of ID as an int! %s", err)
	}

	seqPart, err = strconv.ParseInt(entryIdParts[1], 10, 64)
	if err != nil {
		return fmt.Sprintf("-ERR Could not parse sequence part of ID as an int! %s", err)
	}

	streamAny := r.dataStore.GetOrCreate(streamKey, newRedisStreamAny, 0)
	if stream, ok := streamAny.(redisType_Stream); ok {

		if _, exists := stream.Get(entryId); exists {
			return fmt.Sprintf("-ERR Entry with ID %s already exists in stream! %s", entryId, c.getUsage(ctx))
		}

		fields := stream.GetOrCreate(entryId, newRedisStreamEntry, 0)
		for i := 3; i < len(params); i += 2 {
			field := params[i]
			value := params[i+1]
			fields.PushBack(redisType_FieldValuePair{Field: field, Value: value})
		}

		r.LastEntryId = redisType_EntryId{
			id:  entryId,
			ms:  msPart,
			seq: seqPart,
		}
		return fmt.Sprintf("$%d\r\n%s\r\n", len(entryId), entryId)
	}

	return fmt.Sprintf("-ERR XADD can only be called on streams! %s", c.getUsage(ctx))
}
