package ot

import (
	"errors"
)

type Doc struct {
	content []byte
	userIxs map[uint64]int
	hist    []userOp
}

type userOp struct {
	uid uint64
	op  Op
}

func (d *Doc) Apply(uid uint64, ix int, op Op) (optr Op, err error) {
	minIx, ok := d.userIxs[uid]
	if !ok {
		minIx = -1
	}
	if ix < minIx {
		err = errors.New("Reference for op below user minimum")
		return
	}

	optr = op.Squeeze()
	for i := ix; i < len(d.hist); i++ {
		optr, _, err = transformUsrOp(userOp{uid: uid, op: optr}, d.hist[i])
		if err != nil {
			return
		}
	}
	d.hist = append(d.hist, userOp{uid: uid, op: optr})

	d.userIxs[uid] = len(d.hist) - 1

	return
}

func transformUsrOp(a, b userOp) (at Op, bt Op, err error) {
	if a.uid < b.uid {
		at, bt, err = Transform(a.op, b.op)
		return
	}
	bt, at, err = Transform(b.op, a.op)
	return
}
