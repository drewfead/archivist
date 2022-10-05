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
			`%s  {{  .Name  |  %s  }}{{  ", released on " | %s  }}{{  .PublishDate  |  %s  }}`,
			discIcon,
			focusColor,
			faintColor,
			faintColor,
		),
		Inactive: fmt.Sprintf(
			`    {{  .Name  |  %s  }}{{  ", released on " | %s  }}{{  .PublishDate  |  %s  }}`,
			faintColor,
			faintColor,
			faintColor,
		),
		Selected: fmt.Sprintf(
			`%s  {{  .Name  |  %s  |  %s  }}{{  ", released on " | %s  }}{{  .PublishDate  |  %s  }}`,
			discIcon,
			focusColor,
			highlightColor,
			faintColor,
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
