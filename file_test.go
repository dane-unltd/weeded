package weeded

import (
	"fmt"
	"testing"
	"time"

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
			{Pos: 0, Text: []byte("Hello World!")},
		},
	}

	f.Apply(op, -1)

	fmt.Println(string(f.buf))

	time.Sleep(time.Second)
}
