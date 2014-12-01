package ot

import (
	"bytes"
	"errors"
)

type OpType int8

const (
	Insert (OpType) = 1
	Delete          = -1
)

type Block struct {
	Pos  int64
	Text []byte
}

//One operational transaction
type Operation struct {
	UID    uint64
	OpType OpType
	Blocks []Block //Sortet list with highest Pos first. Non-overlapping in case of Delete.
}

func (o Operation) Inverse() Operation {
	ret := o
	ret.OpType = -ret.OpType
	ret.Blocks = make([]Block, len(o.Blocks))

	cum := int64(0)
	for i := len(o.Blocks) - 1; i >= 0; i-- {
		ret.Blocks[i] = o.Blocks[i]
		ret.Blocks[i].Pos += int64(o.OpType) * cum
		cum += int64(len(o.Blocks[i].Text))
	}

	return ret
}

func (o1 Operation) After(o2 Operation) Operation {
	ret := o1
	ret.Blocks = make([]Block, len(o1.Blocks))
	copy(ret.Blocks, o1.Blocks)

	minPos := o2.Blocks[len(o2.Blocks)-1].Pos

	for i1 := 0; i1 < len(ret.Blocks); i1++ {
		b1 := ret.Blocks[i1]
		switch ret.OpType {
		case Insert:
			if b1.Pos < minPos {
				return ret
			}
			for _, b2 := range o2.Blocks {
				switch o2.OpType {
				case Insert:
					if b1.Pos <= b2.Pos || (b1.Pos == b2.Pos && o1.UID < o2.UID) {
						continue
					}
					ret.Blocks[i1].Pos += int64(len(b2.Text))
				case Delete:
					if b1.Pos <= b2.Pos {
						continue
					}
					if b1.Pos >= b2.Pos+int64(len(b2.Text)) {
						ret.Blocks[i1].Pos -= int64(len(b2.Text))
					} else {
						ret.Blocks[i1].Pos = b2.Pos
					}
				}
			}
			return ret
		case Delete:
			if b1.Pos+int64(len(b1.Text)) <= minPos {
				return ret
			}
			for _, b2 := range o2.Blocks {
				if b1.Pos+int64(len(b1.Text)) <= b2.Pos {
					continue
				}
				switch o2.OpType {
				case Insert:
					if b1.Pos >= b2.Pos {
						ret.Blocks[i1].Pos += int64(len(b2.Text))
					} else {
						//Split Delete operation
						diff := b2.Pos - b1.Pos
						tmp := make([]Block, len(ret.Blocks)+1)
						copy(tmp[:i1], ret.Blocks[:i1])
						copy(tmp[i1+2:], ret.Blocks[i1+1:])
						tmp[i1+1] = Block{Pos: b1.Pos, Text: b1.Text[:diff]}
						tmp[i1] = Block{Pos: b2.Pos + int64(len(b2.Text)), Text: b1.Text[diff:]}
						ret.Blocks = tmp
						b1 = ret.Blocks[i1]
					}
				case Delete:
					if b1.Pos >= b2.Pos+int64(len(b2.Text)) {
						ret.Blocks[i1].Pos -= int64(len(b2.Text))
					} else {
						//Need to remove some of the Text
						left := b2.Pos - b1.Pos
						if left < 0 {
							left = 0
							ret.Blocks[i1].Pos = b2.Pos
						}
						right := b2.Pos + int64(len(b2.Text)) - b1.Pos
						if right > int64(len(b1.Text)) {
							right = int64(len(b1.Text))
						}
						copy(b1.Text[left:], b1.Text[right:])
						ret.Blocks[i1].Text = b1.Text[:int64(len(b1.Text))-(right-left)]
					}
				}
			}
			return ret
		}
	}
	panic("Operation type not specified")
}

func (op Operation) ApplyTo(buf []byte) ([]byte, error) {
	switch op.OpType {
	case Insert:
		if op.Blocks[0].Pos > int64(len(buf)) {
			return nil, errors.New("Insert: index out of range")
		}
		space := int64(cap(buf)) - int64(len(buf))
		cum := int64(0)
		offsets := make([]int64, len(op.Blocks))
		for i := len(op.Blocks) - 1; i >= 0; i-- {
			b := op.Blocks[i]
			cum += int64(len(b.Text))
			offsets[i] = cum
		}
		var ret []byte
		if space < cum {
			ret = make([]byte, int64(len(buf))+cum, (int64(len(buf))+cum)*3/2)
			copy(ret, buf[:op.Blocks[len(op.Blocks)-1].Pos])
		} else {
			ret = buf[:int64(len(buf))+cum]
		}
		prevPos := int64(len(buf))
		for i, b := range op.Blocks {
			offs := offsets[i]
			copy(ret[b.Pos+offs:prevPos+offs], buf[b.Pos:prevPos])
			copy(ret[b.Pos+offs-int64(len(b.Text)):], b.Text)
			prevPos = b.Pos
		}

		return ret, nil
	case Delete:
		if op.Blocks[0].Pos+int64(len(op.Blocks[0].Text)) > int64(len(buf)) {
			return nil, errors.New("Delete: index out of range")
		}
		for _, b := range op.Blocks {
			if bytes.Compare(b.Text, buf[b.Pos:b.Pos+int64(len(b.Text))]) != 0 {
				return nil, errors.New("Delete: Text does not match")
			}
		}
		for _, b := range op.Blocks {
			copy(buf[b.Pos:], buf[b.Pos+int64(len(b.Text)):])
			buf = buf[:len(buf)-len(b.Text)]
		}
		return buf, nil
	}
	return nil, errors.New("Operation type not specified")
}
