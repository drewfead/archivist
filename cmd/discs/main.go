package main

import (
	"context"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/drewfead/archivist/internal/bluray"
	"github.com/drewfead/archivist/internal/output"

	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

type searchConfig struct {
	UseVimBindings bool
}

func search(ctx context.Context, term string, cfg searchConfig) error {
	timeout := 15 * time.Second
	surfCtx, surfCancel := context.WithTimeout(ctx, timeout)
	defer surfCancel()

	stdout := output.Filtered(readline.Stdout, func(bytes []byte) bool {
		return len(bytes) != 1 || bytes[0] != readline.CharBell
	})

	links, err := bluray.Search(surfCtx, term)
	if err != nil {
		return err
	}

	promptTemplates.Help = helpForSearchConfig(cfg)

	prompt := promptui.Select{
		Label:     "Found discs",
		Items:     links,
		Templates: promptTemplates,
		Stdout:    stdout,
		IsVimMode: cfg.UseVimBindings,
		Size:      10,
	}

	selectedIndex, _, err := prompt.Run()
	selectCtx, selectCancel := context.WithTimeout(ctx, timeout)
	defer selectCancel()

	link := links[selectedIndex]
	details, err := bluray.GetDetails(selectCtx, link.BlurayDotComDetailsLink)
	if err != nil {
		return err
	}

	outTemplate, err := template.New("detail").Parse(detailTemplate)
	if err != nil {
		return err
	}
	err = outTemplate.Execute(stdout, details)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "vim",
				Usage: "use vim bindings in interactive mode",
			},
		},
		Name: "discs",
		Action: func(cCtx *cli.Context) error {
			cfg := searchConfig{
				UseVimBindings: cCtx.Bool("vim"),
			}
			return search(cCtx.Context, cCtx.Args().Get(0), cfg)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
