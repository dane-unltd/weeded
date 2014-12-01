package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dane-unltd/weeded/wss"
)

var lg *log.Logger
var root string

func init() {
	lg = log.New(os.Stderr, "Error: ", 0)
}

var weedAddr = "/tmp/weeded.sock"
var weedNetw = "unix"

func main() {
	if len(os.Args) >= 2 {
		strs := strings.SplitN(os.Args[2], ":", 2)
		if len(strs) == 1 {
			weedAddr = strs[0]
		} else {
			weedAddr = strs[1]
			weedNetw = strs[0]
		}
		var err error
		weedAddr, err = filepath.Abs(weedAddr)
		if err != nil {
			lg.Fatalln(err)
		}
	}

	serv := wss.New()
	ln := serv.Listen("tcp", "11000", "file")

	for {
		wsc := ln.Accept()
		conn, err := net.Dial(weedNetw, weedAddr)
		if err != nil {
			lg.Println(err)
			continue
		}

		go func(wsc *wss.Connection, conn net.Conn) {
			enc := json.NewEncoder(conn)
			for {
				msg, err := wsc.Receive()
				if err != nil {
					lg.Println(err)
					return
				}
				switch msg.ID {
				case "ot":
					err := enc.Encode(msg)
					if err != nil {
						lg.Println(err)
						return
					}
				}
			}
		}(wsc, conn)

		go func(wsc *wss.Connection, conn net.Conn) {
			dec := json.NewDecoder(conn)
			for {
				var msg wss.Message
				err := dec.Decode(&msg)
				if err != nil {
					lg.Println(err)
					return
				}
				err = wsc.Send(msg.ID, msg.Data)
				if err != nil {
					lg.Println(err)
					return
				}
			}
		}(wsc, conn)
	}
}
