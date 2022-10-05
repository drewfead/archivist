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
	promptTemplates *promptui.SelectTemplates = &promptui.SelectTemplates{
		Label: "{{  .  }}:",
		Active: fmt.Sprintf(
			"%s  {{  .ShortName  |  %s  }} {{  .PublishDate  |  %s  }}",
			discIcon,
			focusColor,
			faintColor,
		),
		Inactive: fmt.Sprintf(
			"    {{  .ShortName  |  %s  }} {{  .PublishDate  |  %s  }}",
			faintColor,
			faintColor,
		),
		Selected: fmt.Sprintf(
			"%s  {{  .ShortName  |  %s  |  %s  }} {{  .PublishDate  |  %s  }}",
			discIcon,
			focusColor,
			highlightColor,
			faintColor,
		),
	}

	detailTemplate string = `
------- Disc Info -------
Name: {{  .Name  }}
Publisher: {{  .Publisher  }}
Release Year: {{  .ReleaseYear  }}
Runtime: {{  .Runtime  }}
Publish Date: {{  .PublishDate  }}
`
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
