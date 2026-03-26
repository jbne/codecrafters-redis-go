package redistypes

import (
	"github.com/codecrafters-io/redis-starter-go/lib/concurrent"
	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

type (
	List = concurrent.ConcurrentDeque[resptypes.BulkString]
)

func NewList() StoreValue {
	return StoreValue{
		Type: TypeList,
		List: concurrent.NewConcurrentDeque[resptypes.BulkString](),
	}
}
