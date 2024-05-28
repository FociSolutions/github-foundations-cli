/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"gh_foundations/cmd"
)

func main() {
	// if err := ui.Init(); err != nil {
	// 	log.Fatalf("failed to initialize termui: %v", err)
	// }
	// defer ui.Close()

	cmd.Execute()
}
