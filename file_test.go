package weeded

import (
	"fmt"
	"testing"

	"github.com/dane-unltd/weeded/ot"
)

func TestFile(t *testing.T) {
	f, err := NewFile("test.txt")
	if err != nil {
		t.Fatal(err)
	}

	op := ot.Operation{
		OpType: ot.Insert,
		Blocks: []ot.Block{
			{Pos: 0, Text: []byte("Hello World!\n")},
		},
	}

	f.Apply(op, 0)

	op = ot.Operation{
		OpType: ot.Insert,
		Blocks: []ot.Block{
			{Pos: 6, Text: []byte("wide ")},
		},
	}

	f.Apply(op, 1)

	fmt.Println(string(f.Bytes()))
	f.Close()
}
