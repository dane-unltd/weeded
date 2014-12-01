package weeded

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/dane-unltd/msglog"
	"github.com/dane-unltd/weeded/ot"
)

type OtMsg struct {
	Base int64
	Op   ot.Operation
}

type File struct {
	filename string

	otLog    *msglog.Log
	consumer *msglog.Consumer
	buf      []byte
	ots      chan OtMsg
	full     chan io.Writer
	otPos    int64
}

func NewFile(filename string) (*File, error) {
	l, err := msglog.Recover(filename + ".master.weeded")
	fmt.Println(l, err)
	if err != nil {
		return nil, err
	}
	f := &File{
		otLog: l,
	}

	c, err := l.Consumer()
	if err != nil {
		return nil, err
	}

	var op ot.Operation
	for c.HasNext() {
		_, err := c.Next()
		if err != nil {
			return nil, err
		}
		pl, err := c.Payload()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(pl, &op)
		if err != nil {
			return nil, err
		}

		f.buf, err = op.ApplyTo(f.buf)
		if err != nil {
			return nil, err
		}
		f.otPos++
	}
	f.consumer = c
	f.ots = make(chan OtMsg)
	f.full = make(chan io.Writer)
	f.filename = filename

	go f.controller()

	return f, nil
}

func (f *File) controller() {
	defer f.Close()
	c := f.consumer
	for {
		select {
		case otmsg := <-f.ots:
			var err error
			op := otmsg.Op
			if otmsg.Base < f.otPos {
				seq := otmsg.Base + 1
				err = c.Goto(uint64(seq))
				if err != nil {
					log.Println(err)
					return
				}

				var oldop ot.Operation
				for seq <= f.otPos {
					_, err := c.Next()
					if err != nil {
						log.Println(err)
						return
					}
					pl, err := c.Payload()
					if err != nil {
						log.Println(err)
						return
					}
					err = json.Unmarshal(pl, &oldop)
					if err != nil {
						log.Println(err)
						return
					}
					op = op.After(oldop)
					seq++
				}
			}

			f.buf, err = op.ApplyTo(f.buf)
			if err != nil {
				log.Println(err)
				return
			}

			buf, err := json.Marshal(op)
			if err != nil {
				log.Println(err)
				return
			}
			fmt.Println("pushing to log")
			f.otLog.Push(msglog.Msg{From: op.UID}, buf)
			f.otPos++

		case w := <-f.full:
			_, err := w.Write(f.buf)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (f *File) Apply(op ot.Operation, base int64) {
	f.ots <- OtMsg{Op: op, Base: base}
}

func (f *File) Close() {
	f.consumer.Close()
	f.otLog.Close()
	err := ioutil.WriteFile(f.filename, f.buf, 0744)
	if err != nil {
		log.Println(err)
	}
}
