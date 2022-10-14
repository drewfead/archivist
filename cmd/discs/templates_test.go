package main

import (
	"strings"
	"testing"
	"text/template"

	"github.com/drewfead/archivist/internal/bluray"
	"github.com/google/go-cmp/cmp"
	"github.com/manifoldco/promptui"
)

var (
	classicMonsters = bluray.Bluray{
		Name:                    "Universal Classic Monsters: Complete 30-Film Collection Blu-ray",
		ReleaseYear:             "1931-1956",
		Publisher:               "Universal Studios",
		PublishDate:             "Aug 28, 2018",
		PublishCountry:          "United States",
		Runtime:                 "2764 min",
		CollectedMovies:         "Dracula / Drácula / Frankenstein / The Mummy / The Invisible Man / Werewolf of London / Bride of Frankenstein / Dracula's Daughter / Son of Frankenstein / The Invisible Man Returns / The Mummy's Hand / The Invisible Woman / The Wolf Man / The Mummy's Tomb / Ghost of Frankenstein / Invisible Agent / Son of Dracula / Phantom of the Opera / Frankenstein Meets the Wolf Man / The Mummy's Ghost / House of Frankenstein / The Mummy's Curse / The Invisible Man's Revenge / House of Dracula / She-Wolf of London / Abbott and Costello Meet Frankenstein / Abbott & Costello Meet the Invisible Man / Creature from the Black Lagoon / Abbott and Costello Meet the Mummy / Revenge of the Creature / The Creature Walks Among Us",
		BlurayDotComBuyID:       "904168",
		BlurayDotComDetailsLink: "https://www.blu-ray.com/movies/Universal-Classic-Monsters-Complete-30-Film-Collection-Blu-ray/207464/",
		BlurayDotComImageLink:   "https://images.static-bluray.com/movies/covers/207464_medium.jpg",
	}
)

func TestUnit_TemplateFormats(t *testing.T) {
	tests := []struct {
		name     string
		blu      bluray.Bluray
		template string
		expect   string
	}{
		{
			name:     "collected/active",
			blu:      classicMonsters,
			template: PromptTemplates.Active,
			expect:   "💿  \x1b[36mUniversal Classic Monsters: Complete 30-Film Collection Blu-ray\x1b[0m \x1b[2mAug 28, 2018\x1b[0m",
		},
		{
			name:     "collected/details",
			blu:      classicMonsters,
			template: PromptTemplates.Details,
			expect: `
------- Disc Info -------
Name: Universal Classic Monsters: Complete 30-Film Collection Blu-ray
Release Year: 1931-1956
Collects: Dracula / Drácula / Frankenstein / The Mummy / The Invisible Man / Werewolf of London / Bride of Frankenstein / Dracula's Daughter / Son of Frankenstein / The Invisible Man Returns / The Mummy's Hand / The Invisible Woman / The Wolf Man / The Mummy's Tomb / Ghost of Frankenstein / Invisible Agent / Son of Dracula / Phantom of the Opera / Frankenstein Meets the Wolf Man / The Mummy's Ghost / House of Frankenstein / The Mummy's Curse / The Invisible Man's Revenge / House of Dracula / She-Wolf of London / Abbott and Costello Meet Frankenstein / Abbott & Costello Meet the Invisible Man / Creature from the Black Lagoon / Abbott and Costello Meet the Mummy / Revenge of the Creature / The Creature Walks Among Us
Publisher: Universal Studios
Publish Date: Aug 28, 2018
Publish Country: United States
Runtime: 2764 min
Bluray.com: https://www.blu-ray.com/movies/Universal-Classic-Monsters-Complete-30-Film-Collection-Blu-ray/207464/
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp, err := template.New(tt.name).Funcs(promptui.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatal("parsing template", err)
			}
			sb := &strings.Builder{}
			err = tmp.Execute(sb, tt.blu)
			if err != nil {
				t.Fatal("executing template", err)
			}
			if d := cmp.Diff(tt.expect, sb.String()); d != "" {
				t.Error("didn't match expected", d)
			}
		})
	}
}
