// +build ignore

package main

// Example to use readline

import (
	"context"
	"fmt"

	"github.com/mattn/go-colorable"

	"github.com/zetamatta/nyagos/readline"
)

func main() {
	readline1 := readline.Editor{
		Default: "AHAHA",
		Cursor:  3,
		Writer:  colorable.NewColorableStdout(),
	}

	enter_status := 0
	readline.BindKeyClosure(readline.K_CTRL_P,
		func(ctx context.Context, r *readline.Buffer) readline.Result {
			enter_status = -1
			return readline.ENTER
		})

	readline.BindKeyClosure(readline.K_CTRL_N,
		func(ctx context.Context, r *readline.Buffer) readline.Result {
			enter_status = +1
			return readline.ENTER
		})

	text, err := readline1.ReadLine(context.Background())
	fmt.Printf("ENTER_STATUS=%d\n", enter_status)
	if err != nil {
		fmt.Printf("ERR=%s\n", err.Error())
	} else {
		fmt.Printf("TEXT=%s\n", text)
	}
}
