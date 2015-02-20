package ot

import ()

type DocDist struct {
	content  []byte
	userInfo map[uint64]UserInfo
	hist     []userOp
}

type userOp struct {
	uid uint64
	op  Op
}

type UserInfo struct {
	maxIx  int
	lastIx int
	ops    []Op
}

func (d *DocDist) Apply(uid uint64, ix int, op Op) (optr Op, err error) {
	op = op.Squeeze()

	info, ok := d.userInfo[uid]
	if !ok {
		info = UserInfo{maxIx: ix, lastIx: -1}
	}

	var ops []Op
	if ix >= info.maxIx {
		info.lastIx = ix
		info.ops = info.ops[:0]
		info.ops = append(info.ops, op)

	} else {
		info.ops, err = moveUserOps(d.hist, info.ops, info.lastIx, ix, uid)
		if err != nil {
			return
		}
		info.ops = append(info.ops, op)

		ops = make([]Op, len(info.ops))
		copy(ops, info.ops)

		ops, err = moveUserOps(d.hist, ops, ix, info.maxIx, uid)
		if err != nil {
			return
		}

		optr = ops[0]
	}

	for i := ix; i < len(d.hist); i++ {
		_, optr, err = Transform(d.hist[i].op, optr)
		if err != nil {
			return
		}
	}
	d.hist = append(d.hist, userOp{uid: uid, op: optr})

	info.maxIx = len(d.hist) - 1

	d.userInfo[uid] = info

	return
}

func moveUserOps(hist []userOp, ops []Op, from, to int, uid uint64) ([]Op, error) {
	var comb Op
	var err error
	for i := from + 1; i <= to; i++ {
		if hist[i].uid == uid {
			for j := 0; j < len(ops)-1; j++ {
				comb, _, err = Transform(comb, ops[j])
				if err != nil {
					return nil, err
				}
				_, ops[j], err = Transform(comb, ops[j+1])
				if err != nil {
					return nil, err
				}
			}
			ops = ops[:len(ops)-1]
			comb = Op{}
		} else {
			comb, err = Compose(comb, hist[i].op)
			if err != nil {
				return nil, err
			}
		}
	}
	for j := range ops {
		_, ops[j], err = Transform(comb, ops[j])
		if err != nil {
			return nil, err
		}
		comb, _, err = Transform(comb, ops[j])
		if err != nil {
			return nil, err
		}
	}
	return ops, nil
}
