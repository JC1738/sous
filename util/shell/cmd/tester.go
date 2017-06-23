package main

import (
	"fmt"

	"github.com/opentable/sous/util/shell"
)

func main() {
	sh, err := shell.Default()
	if err != nil {
		panic(err)
	}

	std, err := sh.Stderr("pv", "--name", "gotest", "-f", "-t", "-i", "0.0001")
	fmt.Println(std, err)

}
