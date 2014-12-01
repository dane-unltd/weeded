package ot

import (
	"fmt"
	"testing"
)

func TestOT(t *testing.T) {
	b := []byte("Hello World!")

	op := Operation{
		UID:    0,
		OpType: Insert,
		Blocks: []Block{
			{Pos: 6, Text: []byte("wide ")},
			{Pos: 0, Text: []byte("Yo ")},
		},
	}

	op2 := Operation{
		UID:    1,
		OpType: Insert,
		Blocks: []Block{
			{Pos: int64(len(b)), Text: []byte(" and stuff")},
		},
	}

	fmt.Println(string(b))

	var err error
	b, err = op.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	op2 = op2.After(op)

	b, err = op2.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b))

	opinv := op.Inverse()
	opinv = opinv.After(op2)

	b, err = opinv.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	op2inv := op2.Inverse()
	op2inv = op2inv.After(opinv)

	b, err = op2inv.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b))
}
