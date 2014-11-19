package weeded

import (
	"encoding/json"
)

type MsgID string

type Msg struct {
	ID   MsgID
	Data *json.RawMessage
}

type Insert struct {
	Pos  int64
	Text string
}
