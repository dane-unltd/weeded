package ot

import (
	"errors"
)

type Doc struct {
	content []byte
	userIxs map[uint64]int
	hist    []Op
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
		_, optr, err = Transform(d.hist[i], optr)
		if err != nil {
			return
		}
	}
	d.hist = append(d.hist, optr)

	d.userIxs[uid] = len(d.hist) - 1

	return
}
