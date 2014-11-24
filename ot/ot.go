package ot

import (
	"errors"
)

//Represents an operational transformation which is either an Insert or a
//Delete or a list of operations
type Operation interface {
	Apply(buf []byte) ([]byte, error)
	After(ot Operation) Operation
	Inverse() Operation
}

//Op which is a list of operations
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

//Delete operation
type Delete struct {
	SID        int
	Start, End int
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
			ret[1] = Delete{SID: d.SID, Start: d.Start + l, End: d.End - (ot.Pos - d.Start) - l}
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
			ret = ret.After(oti)
		}
		return ret
	}
	panic("unreachable")
	return nil
}

//Insert operation
type Insert struct {
	SID  int
	Pos  int
	Text []byte
}

func (in Insert) Apply(buf []byte) ([]byte, error) {
	space := cap(buf) - len(buf)
	if in.Pos > len(buf) {
		return nil, errors.New("ot Insert: index out of range")
	}
	if space < len(in.Text) {
		ret := make([]byte, (len(buf)+len(in.Text))*3/2)
		copy(ret, buf[:in.Pos])
		copy(ret[in.Pos:], in.Text)
		copy(ret[in.Pos+len(in.Text):], buf[in.Pos:])
		return ret, nil
	}

	buf = buf[:len(buf)+len(in.Text)]

	copy(buf[in.Pos+len(in.Text):], buf[in.Pos:])
	copy(buf[in.Pos:], in.Text)

	return buf, nil
}

func (in Insert) After(ot Operation) Operation {
	switch ot := ot.(type) {
	case Delete:
		ret := in
		if in.Pos <= ot.Start {
			return ret
		}
		if in.Pos >= ot.End {
			ret.Pos -= ot.End - ot.Start
			return ret
		}
		ret.Pos = ot.Start
		return ret
	case Insert:
		if ot.Pos > in.Pos {
			return in
		}
		if ot.Pos == in.Pos && in.SID < ot.SID {
			return in
		}
		ret := in
		ret.Pos = in.Pos + len(ot.Text)
		return ret
	case List:
		var ret Operation
		ret = in
		for _, oti := range ot {
			ret = ret.After(oti)
		}
		return ret
	}
	panic("unreachable")
	return nil
}

func (in Insert) Inverse() Operation {
	return Delete{SID: in.SID, Start: in.Pos, End: in.Pos + len(in.Text), Text: in.Text}
}

//Representation for a text file
type File struct {
	Initial []byte //initial state
	Current []byte //current state
	Hist    List   //list of operations
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

func (f *File) Apply(ot Operation, base int) (int, error) {
	var err error
	for i := base; i < len(f.Hist); i++ {
		ot = ot.After(f.Hist[i])
	}
	f.Current, err = ot.Apply(f.Current)
	if err != nil {
		return 0, err
	}
	f.Hist = append(f.Hist, ot)
	return len(f.Hist) - 1, nil
}
