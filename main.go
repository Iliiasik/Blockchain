package main

import (
	"Blockchain/gui/app"
	"fmt"
)

func main() {
	fmt.Println("=== Blockchain Console ===")
	fmt.Println("This console is used for displaying logs, mining info, and block details.")
	fmt.Println("All operations and events will be shown here.")
	fmt.Println("==========================")

	app := app.NewBlockchainApp()
	app.Run()
}
