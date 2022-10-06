package bluray_test

import (
	"context"
	"testing"

	"github.com/drewfead/archivist/internal/bluray"
)

func TestIntegration_Search(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	out, _ := bluray.Search(context.TODO(), "poltergeist")
	t.Log(out)
}

func TestIntegration_GetDetails(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	out, _ := bluray.GetDetails(context.TODO(), "https://www.blu-ray.com/movies/Poltergeist-4K-Blu-ray/317125/")
	t.Log(out)
}

func TestIntegration_FullSearch(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	out, _ := bluray.FullSearch(context.TODO(), 0, "star")
	t.Log(out)
}
