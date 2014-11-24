package ot

import (
	"fmt"
	"testing"
)

func TestOT(t *testing.T) {
	f := NewFile([]byte("Hello World!"))
	i, err := f.Apply(Insert{SID: 1, Pos: 1, Text: []byte("yo")}, 0)
	fmt.Println(i, err)
	fmt.Println(string(f.Current))
	i, err = f.Apply(f.Hist[0].Inverse(), 1)
	fmt.Println(i, err)
	fmt.Println(string(f.Current))
}
