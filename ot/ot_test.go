package ot

import (
	"fmt"
	"testing"
)

func TestOT(t *testing.T) {
	b := NewBuffer([]byte("Hello World!"))

	op := Operation{
		OpType: Insert,
		Blocks: []Block{
			{Pos: 6, Text: []byte("wide ")},
			{Pos: 0, Text: []byte("Yo ")},
		},
	}

	op2 := Operation{
		OpType: Insert,
		Blocks: []Block{
			{Pos: len(b.Current), Text: []byte(" and stuff")},
		},
	}

	fmt.Println(string(b.Current))

	ix, err := b.Apply(op, -1)
	if err != nil {
		t.Error(err)
	}

	ix2, err := b.Apply(op2, -1)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b.Current))

	_, err = b.Apply(b.Hist[ix].Inverse(), ix)
	if err != nil {
		t.Error(err)
	}

	_, err = b.Apply(b.Hist[ix2].Inverse(), ix2)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b.Current))
}
