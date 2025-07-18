package main

import (
	"Blockchain/controllers"
)

func main() {
	bc := controllers.NewBlockchain()
	defer bc.Db.Close()

	cli := controllers.CLI{Bc: bc}
	cli.Run()
}
