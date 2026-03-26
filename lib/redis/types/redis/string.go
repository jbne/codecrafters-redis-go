package redistypes

import resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"

type (
	String = *resptypes.BulkString
)

func NewString(str string) func() StoreValue {
	return func() StoreValue {
		return StoreValue{
			Type:   TypeString,
			String: resptypes.NewBulkString(str),
		}
	}
}
