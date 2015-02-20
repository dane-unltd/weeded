package ot

import (
	"fmt"
	"testing"
)

func TestOT(t *testing.T) {
	b := []byte("Hello World!")

	op1 := Op{{N: 6}, {N: 5, S: "wide "}, {N: 6}}
	op2 := Op{{N: len(b)}, {N: 10, S: " and stuff"}}

	fmt.Println(string(b))

	b2 := make([]byte, len(b))
	copy(b2, b)

	b, err := op1.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b))

	b2, err = op2.ApplyTo(b2)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b2))

	op1t, op2t, err := Transform(op1, op2)
	if err != nil {
		t.Error(err)
	}

	b, err = op2t.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(b))

	b2, err = op1t.ApplyTo(b2)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(b2))

	op1inv := op1.Inverse()
	op1inv, _, err = Transform(op1inv, op2t)
	if err != nil {
		t.Error(err)
	}

	b, err = op1inv.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	op2inv := op2.Inverse()
	op2inv, _, err = Transform(op2inv, op1inv)
	if err != nil {
		t.Error(err)
	}

	b, err = op2inv.ApplyTo(b)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b))
}
