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

	for i := 10; i < 100000000000000; i = i * 10 {
		fmt.Printf("seq -s ' ' %d\n", i)
		std, err := sh.Stdout("seq", "-s", " ", fmt.Sprintf("%d", i))
		fmt.Println(len(std), err)
	}

}
