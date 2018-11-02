package script

import (
	"testing"
	"fmt"
)

func TestStack(t *testing.T) {
	stack := new(stack)
	stack.PushByteArray([]byte{1,2,3})
	stack.PushByteArray([]byte{4,5,6})
	stack.PushByteArray([]byte{7,8,9})
	end, _ := stack.nipN(1)
	fmt.Println(end)
}