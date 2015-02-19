package ot

import (
	"errors"
)

type SubOp struct {
	N int
	S string
}

type Op []SubOp

func (sop SubOp) IsInsert() bool {
	return sop.N > 0 && len(sop.S) > 0
}

func (sop SubOp) IsDelete() bool {
	return sop.N < 0
}

func (sop SubOp) IsRetain() bool {
	return sop.N > 0 && len(sop.S) == 0
}

func (sop SubOp) IsNoop() bool {
	return sop.N == 0
}

func (op Op) Insert(s string) Op {
	return append(op, SubOp{N: len(s), S: s})
}

func (op Op) Delete(s string) Op {
	return append(op, SubOp{N: -len(s), S: s})
}

func (op Op) Retain(n int) Op {
	return append(op, SubOp{N: n})
}

func (op Op) Count() (ret, del, ins int) {
	for _, sop := range op {
		switch {
		case sop.IsRetain():
			ret += sop.N
		case sop.IsDelete():
			del += -sop.N
		case sop.IsInsert():
			ins += sop.N
		}
	}
	return
}

func (op Op) Equals(other Op) bool {
	if len(op) != len(other) {
		return false
	}
	for i, o := range other {
		if op[i] != o {
			return false
		}
	}
	return true
}

func (op Op) Squeeze() Op {
	var ret Op

	for _, sop := range op {
		if sop.IsNoop() {
			continue
		}
		i := len(ret) - 1
		var lastOp SubOp
		if i > -1 {
			lastOp = ret[i]
		}

		switch {
		case i == -1:
			ret = append(ret, sop)
		case lastOp.IsRetain() && sop.IsRetain():
			ret[i].N += sop.N
		case lastOp.IsDelete() && sop.IsDelete():
			ret[i].N += sop.N
			ret[i].S += sop.S
		case lastOp.IsInsert() && sop.IsInsert():
			ret[i].N += sop.N
			ret[i].S += sop.S
		case lastOp.IsDelete() && sop.IsInsert():
			//insert always before delete
			ret[i] = sop
			ret = append(ret, lastOp)
		default:
			ret = append(ret, sop)
		}
	}
	return ret
}

func (op Op) Inverse() Op {
	inv := make(Op, len(op))
	copy(inv, op)
	for i, sop := range inv {
		if sop.IsInsert() || sop.IsDelete() {
			inv[i].N = -sop.N
		}
	}
	return inv
}

func subop(op Op, i int) (SubOp, int) {
	if i >= 0 && i < len(op) {
		return op[i], i + 1
	}
	return SubOp{}, i
}

func Compose(a, b Op) (Op, error) {
	var ab Op

	a = a.Squeeze()
	b = b.Squeeze()

	reta, _, ins := a.Count()
	retb, del, _ := b.Count()

	if reta+ins != retb+del {
		return ab, errors.New("Compose requires consecutive ops")
	}

	ia, ib := 0, 0

	opa, ia := subop(a, ia)
	opb, ib := subop(b, ib)

	for {
		if opa.IsNoop() && opb.IsNoop() {
			return ab, nil
		}

		if opa.IsDelete() {
			ab = append(ab, opa)

			opa, ia = subop(a, ia)
			continue
		}
		if opb.IsInsert() {
			ab = append(ab, opb)

			opb, ib = subop(b, ib)
			continue
		}

		switch {
		case opa.IsRetain() && opb.IsRetain():
			switch {
			case opa.N > opb.N:
				ab = append(ab, opb)
				opa.N -= opb.N

				opb, ib = subop(b, ib)
			case opa.N == opb.N:
				ab = append(ab, opb)

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N < opb.N:
				ab = append(ab, opa)

				opa, ia = subop(a, ia)
			}
		case opa.IsInsert() && opb.IsDelete():
			switch {
			case opa.N > -opb.N:
				opa.N += opb.N
				opa.S = opa.S[-opb.N:]

				opb, ib = subop(b, ib)
			case opa.N == -opb.N:
				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N < -opb.N:
				opb.N += opa.N
				opb.S = opb.S[opa.N:]

				opa, ia = subop(a, ia)
			}
		case opa.IsInsert() && opb.IsRetain():
			switch {
			case opa.N > opb.N:
				ab = ab.Insert(opa.S[:opb.N])
				opa.N -= opb.N
				opa.S = opa.S[opb.N:]

				opb, ib = subop(b, ib)
			case opa.N == opb.N:
				ab = append(ab, opa)

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N < opb.N:
				ab = append(ab, opa)
				opb.N -= opa.N

				opa, ia = subop(a, ia)
			}
		case opa.IsRetain() && opb.IsDelete():
			switch {
			case opa.N > -opb.N:
				ab = append(ab, opb)
				opa.N += opb.N

				opb, ib = subop(b, ib)
			case opa.N == -opb.N:
				ab = append(ab, opb)

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N < -opb.N:
				ab = ab.Delete(opb.S[:opa.N])
				opb.N += opa.N
				opb.S = opb.S[opa.N:]

				opa, ia = subop(a, ia)
			}
		default:
			panic("unreachable")
		}

	}
	ab = ab.Squeeze()
	return ab, nil
}

func Transform(a, b Op) (at Op, bt Op, err error) {
	ia, ib := 0, 0
	a = a.Squeeze()
	b = b.Squeeze()

	opa, ia := subop(a, ia)
	opb, ib := subop(b, ib)

	for {
		if opa.IsNoop() && opb.IsNoop() {
			return
		}

		if opa.IsInsert() {
			at = append(at, opa)
			bt = bt.Retain(opa.N)

			opa, ia = subop(a, ia)
		}
		if opb.IsInsert() {
			at = at.Retain(opb.N)
			bt = append(bt, opb)

			opb, ib = subop(b, ib)
		}

		switch {
		case opa.IsRetain() && opb.IsRetain():
			minl := 0
			switch {
			case opa.N > opb.N:
				minl = opb.N
				opa.N -= opb.N

				opb, ib = subop(b, ib)
			case opa.N == opb.N:
				minl = opb.N

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N < opb.N:
				minl = opa.N
				opb.N -= opa.N

				opa, ia = subop(a, ia)
			}
			at = at.Retain(minl)
			bt = bt.Retain(minl)
		case opa.IsDelete() && opb.IsDelete():
			switch {
			case opa.N < opb.N:
				opa.N -= opb.N
				opa.S = opa.S[-opb.N:]

				opb, ib = subop(b, ib)
			case opa.N == opb.N:
				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N > opb.N:
				opb.N -= opa.N
				opb.S = opb.S[-opa.N:]

				opa, ia = subop(a, ia)
			}
		case opa.IsDelete() && opb.IsRetain():
			switch {
			case -opa.N > opb.N:
				at = at.Delete(opa.S[:opb.N])
				opa.N += opb.N
				opa.S = opa.S[opb.N:]

				opb, ib = subop(b, ib)
			case -opa.N == opb.N:
				at = append(at, opa)

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case -opa.N < opb.N:
				at = append(at, opa)
				opb.N += opa.N

				opa, ia = subop(a, ia)
			}
		case opa.IsRetain() && opb.IsDelete():
			switch {
			case opa.N < -opb.N:
				bt = bt.Delete(opb.S[:opa.N])
				opb.N += opa.N
				opb.S = opb.S[opa.N:]

				opa, ia = subop(a, ia)
			case opa.N == -opb.N:
				bt = append(bt, opb)

				opa, ia = subop(a, ia)
				opb, ib = subop(b, ib)
			case opa.N > -opb.N:
				bt = append(bt, opb)
				opa.N += opb.N

				opb, ib = subop(b, ib)
			}
		default:
			panic("unreachable")
		}
	}
}

func (op Op) ApplyTo(doc []byte) ([]byte, error) {
	ret, del, ins := op.Count()

	baseLength := ret + del
	targetLength := ret + ins
	workspace := ret + del + ins

	if baseLength != len(doc) {
		return nil, errors.New("The operation's base length must be equal to the documents length.")
	}

	if cap(doc) < workspace {
		tmp := make([]byte, workspace*3/2)
		copy(tmp, doc)
		doc = tmp
	}
	doc = doc[:workspace]

	docIx := 0
	for _, sop := range op {
		switch {
		case sop.IsRetain():
			docIx += sop.N
		case sop.IsInsert():
			copy(doc[docIx+sop.N:], doc[docIx:])
			copy(doc[docIx:], []byte(sop.S))
		case sop.IsDelete():
			if sop.S != string(doc[docIx:docIx-sop.N]) {
				return nil, errors.New("The string which should be deleted does not match the document.")
			}
			copy(doc[docIx:], doc[docIx-sop.N:])
		}
	}
	doc = doc[:targetLength]
	return doc, nil
}
