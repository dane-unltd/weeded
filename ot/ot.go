package ot

import (
	"errors"
)

type Operation interface {
	Apply(buf []byte) ([]byte, error)
	After(ot Operation) Operation
	Inverse() Operation
}

type List []Operation

func (l List) Apply(buf []byte) ([]byte, error) {
	var err error
	for i, op := range l {
		buf, err = op.Apply(buf)
		if err != nil {
			for j := i - 1; j >= 0; j-- {
				buf, _ = op.Inverse().Apply(buf)
			}
			return buf, err
		}
	}
	return buf, nil
}

func (l List) Inverse() Operation {
	inv := make(List, len(l))
	for i := range inv {
		inv[i] = l[len(l)-1-i].Inverse()
	}
	return inv
}

func (l List) After(op Operation) Operation {
	newl := make(List, len(l))
	for i, lop := range l {
		for j := i - 1; j >= 0; j-- {
			lop = lop.After(l[i].Inverse())
		}
		lop = lop.After(op)
		for j := 0; j < i; j++ {
			lop = lop.After(newl[i])
		}
		newl[i] = lop
	}
	for i := 0; i < len(newl); i++ {
		if newl[i] == nil {
			copy(newl[i:], newl[i+1:])
			newl = newl[:len(newl)-1]
			i--
		}
	}
	return newl
}

type Delete struct {
	SID        int64
	Start, End int64
	Text       []byte
}

func (d Delete) Apply(buf []byte) ([]byte, error) {
	if d.Start >= len(buf) || d.End > len(buf) {
		return nil, errors.New("ot Delete: index out of range")
	}
	d.Text = make([]byte, d.End-d.Start)
	copy(d.Text, buf[d.Start:d.End])
	copy(buf[d.Start:], buf[d.End:])
	return buf[:len(buf)-(d.End-d.Start)], nil
}

func (d Delete) Inverse() Operation {
	return Insert{SID: d.SID, Pos: d.Start, Text: d.Text}
}

func (d Delete) After(ot Operation) Operation {
	switch ot := ot.(type) {
	case Delete:
		td := d
		l := ot.End - ot.Start

		if td.Start >= ot.End {
			td.Start -= l
		} else if td.Start >= ot.Start {
			td.Start = ot.Start
		}

		if td.End >= ot.End {
			td.End -= l
		} else if td.End >= ot.Start {
			td.End = ot.Start
		}

		if td.Start == td.End {
			return nil
		}
		return td
	case Insert:
		l := len(ot.Text)
		if ot.Pos > d.Start && ot.Pos < d.End {
			ret := make(List, 2)
			ret[0] = Delete{SID: d.SID, Start: d.Start, End: ot.Pos}
			ret[1] = Delete{SID: d.SID, Start: d.Start + l, End: d.End - (ret[0].End - ret[0].Start) - l}
			return ret
		}
		if ot.Pos <= d.Start {
			return Delete{SID: d.SID, Start: d.Start + l, End: d.End + l}
		}
		return d
	case List:
		var ret Operation
		ret = d
		for _, oti := range ot {
			ret = ret.TransformWith(oti)
		}
		return ret
	}
}

type Insert struct {
	SID  int64
	Pos  int64
	Text []byte
}

func (in Insert) Apply(buf []byte) ([]byte, error) {
}

func (in Insert) After(ot Operation) Operation {
}

func (in Insert) Inverse() Operation {
	return Delete{SID: in.SID, Start: in.Pos, End: in.Pos + len(in.Text), Text: in.Text}
}

type File struct {
	Initial []byte
	Current []byte
	Hist    List
}

func NewFile(buf []byte) *File {
	current := make([]byte, len(buf))
	copy(current, buf)
	return &File{
		Initial: buf,
		Current: current,
		Hist:    List{},
	}
}

func (f *File) Apply(ot Operation, base int64) (int64, error) {
	var err error
	f.Current, err = ot.Apply(f.Current)
	if err != nil {
		return 0, err
	}
	f.Hist = append(f.Hist, ot)
	return int64(len(f.Hist) - 1), nil
}
