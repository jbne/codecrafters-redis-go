package redistypes

import (
	"github.com/codecrafters-io/redis-starter-go/lib/concurrent"
)

type (
	StoreKey       = string
	StoreValueType int
	StoreValue     struct {
		Type   StoreValueType
		String String
		List   List
		Stream ConcurrentStream
	}

	DataStore interface {
		concurrent.ConcurrentMap[StoreKey, StoreValue]
	}
)

const (
	TypeUnknown StoreValueType = iota
	TypeString
	TypeList
	TypeStream
)

func NewRedisDataStore() DataStore {
	return concurrent.NewConcurrentMap[StoreKey, StoreValue]()
}
