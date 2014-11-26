package main

import (
	"encoding/json"
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

func main() {
	addr := "/tmp/weeded.sock"
	netw := "unix"
	if len(os.Args) >= 3 {
		strs := strings.Split(os.Args[2], ":")
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

	ln, err := net.Listen(netw, addr)
	if err != nil {
		lg.Fatalln(err)
	}

	for {
		var uid int
		conn, err := ln.Accept()
		if err != nil {
			lg.Println(err)
		}
		uid++
		go handleClient(conn, uid)
	}
}

func handleClient(conn net.Conn, uid int) {
	dec := json.NewDecoder(conn)
	for {
		var msg weeded.Msg
		err := dec.Decode(&msg)
		if err != nil {
			lg.Println(err)
			return
		}
		switch msg.ID {
		case "op":
			var op ot.Operation
			err := json.Unmarshal(*msg.Data, &op)
			op.UID = uid
			if err != nil {
				lg.Println(err)
				return
			}
		}
	}
}
