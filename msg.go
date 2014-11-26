package weeded

import (
	"encoding/json"
)

type MsgID string

type Msg struct {
	ID   MsgID
	Data *json.RawMessage
}
