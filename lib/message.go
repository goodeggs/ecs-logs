package lib

import (
	"bytes"
	"encoding/json"
	"sync"

	"github.com/segmentio/ecs-logs-go"
	"github.com/segmentio/jutil"
)

type Message struct {
	Group  string                 `json:"group,omitempty"`
	Stream string                 `json:"stream,omitempty"`
	Event  ecslogs.Event          `json:"event,omitempty"`
	JSON   map[string]interface{} `json:"json,omitempty"`
}

func jsonMarshalWithTimeFirst(m map[string]interface{}) ([]byte, error) {
	if t, ok := m["time"]; ok {
		mWithoutTime := make(map[string]interface{})
		for k, v := range m {
			if k == "time" {
				continue
			}
			mWithoutTime[k] = v
		}

		b, err := json.Marshal(mWithoutTime)
		if err != nil {
			return nil, err
		}

		tb, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}

		res := bytes.Join([][]byte{b[:1], []byte(`"time":`), tb, []byte(`,`), b[1:]}, []byte{})
		return res, nil
	}

	return json.Marshal(m)
}

func (m Message) Bytes() []byte {
	if m.JSON != nil {
		b, _ := jsonMarshalWithTimeFirst(m.JSON)
		return b
	}
	b, _ := json.Marshal(m)
	return b
}

func (m Message) String() string {
	return string(m.Bytes())
}

func (m Message) ContentLength() int {
	if m.JSON != nil {
		n, _ := jutil.Length(m.JSON)
		return n
	}
	n, _ := jutil.Length(m.Event)
	return n
}

type MessageBatch []Message

func (list MessageBatch) Swap(i int, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list MessageBatch) Less(i int, j int) bool {
	return list[i].Event.Time.Before(list[j].Event.Time)
}

func (list MessageBatch) Len() int {
	return len(list)
}

type MessageQueue struct {
	C      <-chan struct{}
	signal chan struct{}
	mutex  sync.Mutex
	batch  MessageBatch
}

func NewMessageQueue() *MessageQueue {
	c := make(chan struct{}, 1)
	return &MessageQueue{
		C:      c,
		signal: c,
		batch:  make(MessageBatch, 0, 100),
	}
}

func (q *MessageQueue) Push(msg Message) {
	q.mutex.Lock()
	q.batch = append(q.batch, msg)
	q.mutex.Unlock()
}

func (q *MessageQueue) Notify() {
	select {
	default:
	case q.signal <- struct{}{}:
	}
}

func (q *MessageQueue) Flush() (batch MessageBatch) {
	q.mutex.Lock()
	batch = make(MessageBatch, len(q.batch))
	copy(batch, q.batch)
	q.batch = q.batch[:0]
	q.mutex.Unlock()
	return
}
