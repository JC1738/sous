package main

import (
	"fmt"
	"os"

	"github.com/opentable/sous/util/shell"
)

func main() {
	sh, err := shell.Default()
	if err != nil {
		panic(err)
	}
	sh.TeeEcho = os.Stdout
	sh.LongRunning(true)

	std, err := sh.Stdout("docker", "build", "--pull", "service")
	fmt.Println(std, err)

}
