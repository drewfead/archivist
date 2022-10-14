package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

const (
	discIcon       = "\U0001F4BF"
	highlightColor = "red"
	focusColor     = "cyan"
	faintColor     = "faint"
)

var (
	PromptTemplates *promptui.SelectTemplates = &promptui.SelectTemplates{
		Label: "{{  .  }}:",
		Active: fmt.Sprintf(
			"%s {{  .Name  |  %s  }} {{  .PublishDate  |  %s  }}",
			discIcon,
			focusColor,
			faintColor,
		),
		Inactive: fmt.Sprintf(
			"   {{  .Name  |  %s  }} {{  .PublishDate  |  %s  }}",
			faintColor,
			faintColor,
		),
		Selected: fmt.Sprintf(
			"%s {{  .Name  |  %s  |  %s  }} {{  .PublishDate  |  %s  }}",
			discIcon,
			focusColor,
			highlightColor,
			faintColor,
		),
		Details: `
------- Disc Info -------
Name: {{  .Name  }}
Release Year: {{  .ReleaseYear  }}
Collects: {{  .CollectedMovies  }}
Publisher: {{  .Publisher  }}
Publish Date: {{  .PublishDate  }}
Publish Country: {{  .PublishCountry  }}
Runtime: {{  .Runtime  }}
Bluray.com: {{  .BlurayDotComDetailsLink  }}
`,
	}
)

const stdNav = "Use the arrow keys to navigate: ↓ ↑ → ←"
const vimNav = "use the vim keys to navigate: j k l h"

func helpForSearchConfig(cfg searchConfig) string {
	if cfg.UseVimBindings {
		return vimNav
	} else {
		return stdNav
	}
}
