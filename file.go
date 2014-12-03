package weeded

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	"github.com/dane-unltd/msglog"
	"github.com/dane-unltd/weeded/ot"
)

type OtMsg struct {
	Ix int64
	Op ot.Operation
}

type File struct {
	filename string

	otLog    *msglog.Log
	consumer *msglog.Consumer
	buf      []byte
	ots      chan OtMsg
	full     chan io.Writer
	quit     chan chan struct{}
	nextIx   int64
}

func NewFile(filename string) (*File, error) {
	l, err := msglog.Recover(filename + ".master.weeded")
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
		f.nextIx++
	}
	f.consumer = c
	f.ots = make(chan OtMsg)
	f.full = make(chan io.Writer)
	f.quit = make(chan chan struct{})
	f.filename = filename

	go f.controller()

	return f, nil
}

func (f *File) controller() {
	c := f.consumer
	for {
		select {
		case otmsg := <-f.ots:
			var err error
			op := otmsg.Op
			if otmsg.Ix < f.nextIx {
				seq := otmsg.Ix
				err = c.Goto(uint64(seq))
				if err != nil {
					log.Println(err)
					f.closeAll()
					return
				}

				var oldop ot.Operation
				for seq < f.nextIx {
					_, err := c.Next()
					if err != nil {
						log.Println(err)
						f.closeAll()
						return
					}
					pl, err := c.Payload()
					if err != nil {
						log.Println(err)
						f.closeAll()
						return
					}
					err = json.Unmarshal(pl, &oldop)
					if err != nil {
						log.Println(err)
						f.closeAll()
						return
					}
					op = op.After(oldop)
					seq++
				}
			}

			f.buf, err = op.ApplyTo(f.buf)
			if err != nil {
				log.Println(err)
				f.closeAll()
				return
			}

			buf, err := json.Marshal(op)
			if err != nil {
				log.Println(err)
				f.closeAll()
				return
			}
			f.otLog.Push(msglog.Msg{From: op.UID}, buf)
			f.nextIx++

		case w := <-f.full:
			_, err := w.Write(f.buf)
			if err != nil {
				log.Println(err)
			}

		case ret := <-f.quit:
			f.closeAll()
			ret <- struct{}{}
			return
		}
	}
}

func (f *File) Apply(op ot.Operation, ix int64) {
	f.ots <- OtMsg{Op: op, Ix: ix}
}

func (f *File) Close() {
	ret := make(chan struct{})
	f.quit <- ret
	<-ret
}

func (f *File) closeAll() {
	f.consumer.Close()
	f.otLog.Close()
	err := ioutil.WriteFile(f.filename, f.buf, 0744)
	if err != nil {
		log.Println(err)
	}
}
