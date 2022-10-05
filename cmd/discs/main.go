package main

import (
	"context"
	"errors"
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
	Interactive    bool
	UseVimBindings bool
	Timeout        time.Duration
}

var (
	ErrNoSelectableResults = errors.New("no selectable results")
)

func search(ctx context.Context, term string, cfg searchConfig) error {
	surfCtx, surfCancel := context.WithTimeout(ctx, cfg.Timeout)
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
	selectCtx, selectCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer selectCancel()

	if len(links) <= selectedIndex {
		return ErrNoSelectableResults
	}

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
		Commands: []*cli.Command{
			{
				Name:    "search",
				Aliases: []string{"s"},
				Usage:   "search for discs",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "vim",
						Value:   false,
						Usage:   "use vim bindings where applicable",
						EnvVars: []string{"DISCS_VIM_MODE"},
					},
					&cli.BoolFlag{
						Name:    "interactive",
						Value:   true,
						Usage:   "use interactive mode for search. gives access to extra data",
						Aliases: []string{"i"},
					},
					&cli.DurationFlag{
						Name:  "timeout",
						Value: 15 * time.Second,
						Usage: "set the timeout after which the CLI will fail",
					},
				},
				Action: func(cCtx *cli.Context) error {
					cfg := searchConfig{
						UseVimBindings: cCtx.Bool("vim"),
						Interactive:    cCtx.Bool("interactive"),
						Timeout:        cCtx.Duration("timeout"),
					}
					return search(cCtx.Context, cCtx.Args().Get(0), cfg)
				},
			},
		},
		Name: "discs",
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
