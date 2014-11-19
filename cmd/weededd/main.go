package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dane-unltd/weeded"
)

var lg *log.Logger
var root string

func init() {
	lg = log.New(os.Stderr, "Error: ", 0)
}

func main() {
	addr := "/tmp/auth.sock"
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
		conn, err := ln.Accept()
		if err != nil {
			lg.Println(err)
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	dec := json.NewDecoder(conn)
	for {
		var msg weeded.Msg
		err := dec.Decode(&msg)
		if err != nil {
			lg.Println(err)
			return
		}
		switch msg.ID {
		case "insert":
			var ins weeded.Insert
			err := json.Unmarshal(*msg.Data, &ins)
			if err != nil {
				lg.Println(err)
				return
			}
		}
	}
}
