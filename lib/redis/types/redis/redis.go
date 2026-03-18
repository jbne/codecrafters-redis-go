package redistypes

import (
	"github.com/codecrafters-io/redis-starter-go/lib/concurrent"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	List = concurrent.ConcurrentDeque[resptypes.BulkString]

	// Entry ID => [field => value, ...]
	Stream        = concurrent.ConcurrentMap[string, StreamEntries]
	StreamEntries = concurrent.ConcurrentDeque[StreamEntry]
	StreamEntry   = resptypes.Array[resptypes.BulkString]

	StreamEntryId struct {
		Id  string
		Ms  int64
		Seq int64
	}
)

func NewStreamEntries() StreamEntries {
	return concurrent.NewConcurrentDeque[StreamEntry]()
}

func NewStream() any {
	return concurrent.NewConcurrentMap[string, StreamEntries]()
}

func NewList() any {
	return concurrent.NewConcurrentDeque[resptypes.BulkString]()
}
