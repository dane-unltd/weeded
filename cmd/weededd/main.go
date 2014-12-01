package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dane-unltd/msglog"
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
	if len(os.Args) >= 2 {
		strs := strings.SplitN(os.Args[1], ":", 2)
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

	aq := make(chan *Aquire)

	go manageBuffers(aq)

	for {
		var uid int
		conn, err := ln.Accept()
		if err != nil {
			lg.Println(err)
		}
		uid++
		go handleClient(conn, uid, aq)
	}
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

type Buffer struct {
	b          *ot.Buffer
	ots        chan *ot.Operation
	connect    chan Conn
	disconnect chan Conn
	nUsers     int
}

func NewBuffer(file string) (*Buffer, error) {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return &Buffer{
		b:          ot.NewBuffer(f),
		ots:        make(chan *ot.Operation),
		connect:    make(chan Conn),
		disconnect: make(chan Conn),
	}, nil
}

func (b *Buffer) Run() {
	users := make(map[int]Conn)
	for {
		select {
		case op := <-b.ots:
			b.b.Apply(*op)
		case conn := <-b.connect:
			users[conn.uid] = conn
			conn.Send("buffer", string(b.b.Current))
		case conn := <-b.disconnect:
			delete(users, conn.uid)
		}
	}
}

func (b *Buffer) Apply(op *ot.Operation) {
	b.ots <- op
}

type Aquire struct {
	f    *string
	conn Conn
	ret  chan<- *Buffer
}

func manageBuffers(aq chan *Aquire) {
	buffers := make(map[string]*Buffer)
	files := make(map[int]string)

	for {
		req := <-aq

		if req.f == nil {
			name, ok := files[req.conn.uid]
			if !ok {
				continue
			}
			buf, ok := buffers[name]
			if !ok {
				continue
			}
			buf.nUsers--
			buf.disconnect <- req.conn
			delete(files, req.conn.uid)

			fmt.Println("disconnecting:", req.conn.uid)
			if buf.nUsers == 0 {
				delete(buffers, name)
				err := ioutil.WriteFile(name, buf.b.Current, 0744)
				if err != nil {
					lg.Println(err)
				}
			}
			continue
		}

		buf, ok := buffers[*req.f]

		if !ok {
			var err error
			buf, err = NewBuffer(*req.f)
			if err != nil {
				lg.Println(err)
				continue
			}

			go buf.Run()
			buffers[*req.f] = buf
		}
		files[req.conn.uid] = *req.f

		buf.nUsers++
		buf.connect <- req.conn
		req.ret <- buf
	}
}

func handleClient(conn net.Conn, uid int, aq chan *Aquire) {
	dec := json.NewDecoder(conn)
	wconn := Conn{uid: uid, enc: json.NewEncoder(conn)}
	var buf *Buffer

	defer func() { aq <- &Aquire{conn: wconn} }()

	for {
		var msg weeded.Msg
		err := dec.Decode(&msg)
		if err != nil {
			lg.Println(err)
			return
		}
		switch msg.ID {
		case "ot":
			var op ot.Operation
			err := json.Unmarshal(*msg.Data, &op)
			op.UID = uid
			if err != nil {
				lg.Println(err)
				return
			}
			if buf != nil {
				buf.Apply(&op)
			}
		case "open":
			var f string
			err := json.Unmarshal(*msg.Data, &f)
			if err != nil {
				lg.Println(err)
				return
			}
			ret := make(chan (*Buffer))
			aq <- &Aquire{f: &f, conn: wconn, ret: ret}
			buf = <-ret
		}
	}
}
