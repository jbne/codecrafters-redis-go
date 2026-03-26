package redistypes

import (
	"fmt"
	"math"
	"sync"
	"time"

	resptypes "github.com/codecrafters-io/redis-starter-go/lib/redis/types/resp"
)

const (
	// This is the max possible sequence number that Redis supports.
	MaxSequenceNum = uint64(math.MaxUint64)
)

type (
	// Entry ID => [field => value, ...]
	StreamEntryId struct {
		Ms  uint64
		Seq uint64
	}

	streamEntry struct {
		StreamEntryId
		resptypes.Array[resptypes.BulkString]
	}

	stream struct {
		mu                sync.RWMutex
		lastStreamEntryId StreamEntryId
		entries           resptypes.Array[streamEntry]
	}

	AddStreamEntryId struct {
		StreamEntryId
		GenMs  bool
		GenSeq bool
	}

	ConcurrentStream interface {
		AddEntry(id AddStreamEntryId, entry resptypes.Array[resptypes.BulkString]) resptypes.RespSerializable
		GetEntries(start StreamEntryId, end StreamEntryId) resptypes.RespSerializable
	}
)

func NewStream() StoreValue {
	return StoreValue{
		Type: TypeStream,
		Stream: &stream{
			lastStreamEntryId: StreamEntryId{},
			entries:           make(resptypes.Array[streamEntry], 0),
		},
	}
}

func (s StreamEntryId) ToRespString() string {
	str := resptypes.NewBulkString(fmt.Sprintf("%d-%d", s.Ms, s.Seq))
	return str.ToRespString()
}

func (s streamEntry) ToRespString() string {
	entry := make(resptypes.Array[resptypes.RespSerializable], 2)
	entry[0] = s.StreamEntryId
	entry[1] = s.Array
	return entry.ToRespString()
}

func (s *stream) AddEntry(id AddStreamEntryId, entry resptypes.Array[resptypes.BulkString]) resptypes.RespSerializable {
	if id.GenMs {
		id.Ms = uint64(time.Now().UnixMilli())
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if id.GenSeq {
		if id.Ms == s.lastStreamEntryId.Ms {
			id.Seq = s.lastStreamEntryId.Seq + 1
		}
	}

	if id.Ms < s.lastStreamEntryId.Ms {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")}
	}

	if id.Ms == s.lastStreamEntryId.Ms && id.Seq <= s.lastStreamEntryId.Seq {
		return resptypes.SimpleError{Val: fmt.Errorf("ERR The ID specified in XADD is equal or smaller than the target stream top item")}
	}

	newId := StreamEntryId{
		Ms:  id.Ms,
		Seq: id.Seq,
	}

	s.entries = append(s.entries, streamEntry{
		StreamEntryId: newId,
		Array:         entry,
	})
	s.lastStreamEntryId = newId

	return resptypes.NewBulkString(fmt.Sprintf("%d-%d", newId.Ms, newId.Seq))
}

func (s *stream) GetEntries(start StreamEntryId, end StreamEntryId) resptypes.RespSerializable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make(resptypes.Array[resptypes.RespSerializable], 0)
	for _, entry := range s.entries {
		if entry.Ms > end.Ms {
			break
		}

		if entry.Ms == end.Ms && entry.Seq > end.Seq {
			break
		}

		if entry.Ms < start.Ms {
			continue
		}

		if entry.Ms == start.Ms && entry.Seq < start.Seq {
			continue
		}

		entries = append(entries, entry)
	}

	return entries
}
