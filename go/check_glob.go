package main

import (
	"fmt"
	"github.com/gobwas/glob"
)

func main() {
	g, _ := glob.Compile("*")
	fmt.Printf("%T\n", g)
}
