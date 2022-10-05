package main

import (
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/drewfead/archivist/internal/bluray"
	"github.com/drewfead/archivist/internal/output"

	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
)

func main() {
	timeout := 15 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout := output.Filtered(readline.Stdout, func(bytes []byte) bool {
		return len(bytes) != 1 || bytes[0] != readline.CharBell
	})

	searchTerm := os.Args[1]
	if searchTerm == "" {
		os.Exit(1)
	}

	links, err := bluray.Search(ctx, searchTerm)
	if err != nil {
		fmt.Printf("failed %v\n", err)
		return
	}

	prompt := promptui.Select{
		Label:     "Found discs",
		Items:     links,
		Templates: promptTemplates,
		Stdout:    stdout,
	}

	selectedIndex, _, err := prompt.Run()
	link := links[selectedIndex]
	details, err := bluray.GetDetails(ctx, link.BlurayDotComDetailsLink)
	if err != nil {
		fmt.Printf("failed %v\n", err)
		return
	}

	outTemplate, err := template.New("detail").Parse(detailTemplate)
	if err != nil {
		fmt.Printf("failed %v\n", err)
	}
	err = outTemplate.Execute(stdout, details)
	if err != nil {
		fmt.Printf("failed %v\n", err)
	}
}
