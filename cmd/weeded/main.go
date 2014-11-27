package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dane-unltd/weeded"
	"github.com/dane-unltd/weeded/ot"
)

var lg *log.Logger
var root string

func init() {
	lg = log.New(os.Stderr, "Error: ", 0)
}

type Encoder interface {
	Encode(v interface{}) error
}

type Conn struct {
	uid int
	enc Encoder
}

func (c Conn) Send(id weeded.MsgID, data interface{}) error {
	return c.enc.Encode(struct {
		ID   weeded.MsgID
		Data interface{}
	}{id, data})
}

func main() {
	addr := "/tmp/weeded.sock"
	netw := "unix"
	if len(os.Args) >= 2 {
		strs := strings.SplitN(os.Args[2], ":", 2)
		if len(strs) == 1 {
			addr = strs[0]
		} else {
			addr = strs[1]
			netw = strs[0]
		}
		var err error
		addr, err = filepath.Abs(addr)
		if err != nil {
			lg.Fatalln(err)
		}
	}

	conn, err := net.Dial(netw, addr)
	if err != nil {
		lg.Fatalln(err)
	}

	go input(conn)

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	out := Conn{0, enc}

	out.Send("open", "/Users/david_neumann/test.txt")
	for {
		var msg weeded.Msg
		err := dec.Decode(&msg)
		if err != nil {
			lg.Println(err)
			return
		}
		switch msg.ID {
		case "buffer":
			fmt.Println(string(*msg.Data))

			op := ot.Operation{
				Base:   -1,
				OpType: ot.Insert,
				Blocks: []ot.Block{{Pos: 3, Text: []byte("blabla")}},
			}

			out.Send("ot", op)
		}
	}
}

func input(conn net.Conn) {
}
