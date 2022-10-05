package bluray

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/robertkrimen/otto"
)

var (
	ErrOperationTimeout    = errors.New("operation timed out")
	ErrJavascriptEvaluate  = errors.New("failed to evaluate javascript")
	ErrJavascriptTranslate = errors.New("failed to translate javascript data")
	ErrBadResponseStatus   = errors.New("received bad response status")
	ErrInsufficientDetails = errors.New("couldn't find sufficient details")
)

type searchMessage struct {
	EndSignal               bool
	Index                   int
	Display                 string
	BlurayDotComID          string
	BlurayDotComDetailsLink string
	BlurayDotComImageLink   string
}

func mergeSearch(a, b searchMessage) searchMessage {
	out := searchMessage{}
	if a.EndSignal || b.EndSignal {
		out.EndSignal = true
	}
	if a.Display != "" {
		out.Display = a.Display
	} else if b.Display != "" {
		out.Display = b.Display
	}
	if a.BlurayDotComID != "" {
		out.BlurayDotComID = a.BlurayDotComID
	} else if b.BlurayDotComID != "" {
		out.BlurayDotComID = b.BlurayDotComID
	}
	if a.BlurayDotComDetailsLink != "" {
		out.BlurayDotComDetailsLink = a.BlurayDotComDetailsLink
	} else if b.BlurayDotComDetailsLink != "" {
		out.BlurayDotComDetailsLink = b.BlurayDotComDetailsLink
	}
	if a.BlurayDotComImageLink != "" {
		out.BlurayDotComImageLink = a.BlurayDotComImageLink
	} else if b.BlurayDotComImageLink != "" {
		out.BlurayDotComImageLink = b.BlurayDotComImageLink
	}
	return out
}

func collectSearch(msgs []searchMessage) []searchMessage {
	var out []searchMessage

	for _, msg := range msgs {
		if len(out) <= msg.Index {
			addEmpty := msg.Index - len(out) + 1
			for i := 0; i < addEmpty; i++ {
				out = append(out, searchMessage{})
			}
		}
		out[msg.Index] = mergeSearch(out[msg.Index], msg)
	}

	return out
}

func normalizeSearchDisplay(display string) (string, string) {
	parts := strings.Split(display, "\u00a0") // non-breaking space
	if len(parts) > 1 {
		return parts[1], parts[0]
	}
	return parts[0], ""
}

func transformSearch(msg searchMessage) (BlurayLinks, bool) {
	name, date := normalizeSearchDisplay(msg.Display)
	out := BlurayLinks{
		ShortName:               name,
		PublishDate:             date,
		BlurayDotComBuyID:       msg.BlurayDotComID,
		BlurayDotComDetailsLink: msg.BlurayDotComDetailsLink,
		BlurayDotComImageLink:   msg.BlurayDotComImageLink,
	}

	return out, out.ShortName != "" || out.BlurayDotComBuyID != "" || out.BlurayDotComDetailsLink != ""
}

func transformAllSearch(msgs []searchMessage) []BlurayLinks {
	out := make([]BlurayLinks, 0, len(msgs))
	for _, msg := range msgs {
		if blu, ok := transformSearch(msg); ok {
			out = append(out, blu)
		}
	}
	return out
}

type BlurayLinks struct {
	ShortName               string
	PublishDate             string
	BlurayDotComBuyID       string
	BlurayDotComDetailsLink string
	BlurayDotComImageLink   string
}

type BlurayDetails struct {
	Name        string
	Publisher   string
	ReleaseYear string
	Runtime     string
	PublishDate string
}

type Bluray struct {
	BlurayDetails
	BlurayLinks
}

func largestSize(arrs ...[]string) int {
	largestSize := 0
	for _, arr := range arrs {
		if len(arr) > largestSize {
			largestSize = len(arr)
		}
	}
	return largestSize
}

func Search(ctx context.Context, searchTerm string) ([]BlurayLinks, error) {
	quicksearch := colly.NewCollector()
	quicksearchJS := otto.New()
	errorChannel := make(chan error)
	msgChannel := make(chan searchMessage)

	quicksearch.OnRequest(func(r *colly.Request) {
		r.Headers.Set("X-Requested-With", "XMLHttpRequest")
		r.Headers.Set("Referer", "https://www.blu-ray.com")
		r.Headers.Set("Cookie", "pw_bottom_filter=blur; firstview=1; search_section=bluraymovies")
	})

	quicksearch.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			errorChannel <- fmt.Errorf("got status code %d from blu-ray.com quicksearch: %w", r.StatusCode, ErrBadResponseStatus)
			return
		}
	})

	// todo: really brittle js evaluation, figure out a better way
	evalJSArray := func(varName string) []string {
		var out []string

		jsVar, err := quicksearchJS.Get(varName)
		if err != nil {
			errorChannel <- fmt.Errorf("finding %s from blu-ray.com quicksearch: %w", varName, ErrJavascriptEvaluate)
			return nil
		}

		untyped, _ := jsVar.Export()
		switch jsVar.Class() {
		case "Array":
			if typed, ok := untyped.([]string); ok {
				out = append(out, typed...)
			}
		case "String":
			out = append(out, untyped.(string))
		default:
			errorChannel <- fmt.Errorf("translating %s of type %s from blu-ray.com javascript: %w", varName, jsVar.Class(), ErrJavascriptTranslate)
		}

		return out
	}

	// search hit links
	quicksearch.OnHTML("script", func(h *colly.HTMLElement) {
		for _, jsLine := range strings.Split(h.Text, ";") {
			trimmed := strings.TrimSpace(jsLine)
			termed := fmt.Sprintf("%s;", trimmed)
			if strings.HasPrefix(termed, "var") || strings.HasPrefix(termed, "const") {
				_, err := quicksearchJS.Run(termed)
				if err != nil {
					errorChannel <- fmt.Errorf("running js from blu-ray.com quicksearch: %w", ErrJavascriptEvaluate)
					return
				}
			}
		}

		ids := evalJSArray("ids")
		urls := evalJSArray("urls")
		images := evalJSArray("images")
		size := largestSize(ids, urls, images)

		i := 0
		for i < size {
			var id string
			var url string
			var image string

			if len(ids) > i {
				id = ids[i]
			}

			if len(urls) > i {
				url = urls[i]
			}

			if len(images) > i {
				image = images[i]
			}

			msgChannel <- searchMessage{
				Index:                   i,
				BlurayDotComID:          id,
				BlurayDotComDetailsLink: url,
				BlurayDotComImageLink:   image,
			}
			i++
		}

		msgChannel <- searchMessage{
			EndSignal: true,
			Index:     i + 1,
		}
	})

	// search hit display text
	quicksearch.OnHTML("ul", func(h *colly.HTMLElement) {
		for i, text := range h.ChildTexts("li") {
			msgChannel <- searchMessage{
				Display: text,
				Index:   i,
			}
		}
		msgChannel <- searchMessage{
			EndSignal: true,
		}
	})

	mergingDatasetCount := 2 // search hit links, search hit display text
	// timeout
	go func() {
		<-ctx.Done()
		for i := 0; i < mergingDatasetCount; i++ {
			msgChannel <- searchMessage{EndSignal: true}
		}
	}()

	go func() {
		request := make(map[string]string, 4)
		request["section"] = "bluraymovies"
		request["userid"] = "-1"
		request["country"] = "all"
		request["keyword"] = searchTerm

		quicksearch.Post("https://www.blu-ray.com/search/quicksearch.php", request)
	}()

	var msgs []searchMessage
	endSignals := 0
	for {
		select {
		// todo: accumulate errors instead of breaking on first
		case err := <-errorChannel:
			return nil, err
		case msg := <-msgChannel:
			if msg.EndSignal {
				endSignals++
			}
			msgs = append(msgs, msg)
			if endSignals >= mergingDatasetCount {
				return transformAllSearch(collectSearch(msgs)), nil
			}
		}
	}
}

type detailsMessage struct {
	EndSignal   bool
	Name        string
	Publisher   string
	PublishDate string
	Runtime     string
	ReleaseYear string
}

func reduceDetails(msgs []detailsMessage) detailsMessage {
	out := detailsMessage{}

	for _, msg := range msgs {
		if msg.EndSignal {
			out.EndSignal = true
		}
		if msg.Name != "" {
			out.Name = msg.Name
		}
		if msg.PublishDate != "" {
			out.PublishDate = msg.PublishDate
		}
		if msg.Publisher != "" {
			out.Publisher = msg.Publisher
		}
		if msg.ReleaseYear != "" {
			out.ReleaseYear = msg.ReleaseYear
		}
		if msg.Runtime != "" {
			out.Runtime = msg.Runtime
		}
	}

	return out
}

func transformDetails(msg detailsMessage) (*BlurayDetails, bool) {
	out := &BlurayDetails{
		Name:        msg.Name,
		Publisher:   msg.Publisher,
		PublishDate: msg.PublishDate,
		ReleaseYear: msg.ReleaseYear,
		Runtime:     msg.Runtime,
	}

	return out, out.Name != ""
}

func GetDetails(ctx context.Context, detailsLink string) (*BlurayDetails, error) {
	details := colly.NewCollector()
	errorChannel := make(chan error)
	msgChannel := make(chan detailsMessage)

	details.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Referer", "https://www.blu-ray.com")
		r.Headers.Set("Cookie", "pw_bottom_filter=blur; firstview=1; search_section=bluraymovies")
	})

	details.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			errorChannel <- fmt.Errorf("got status code %d from blu-ray.com quicksearch: %w", r.StatusCode, ErrBadResponseStatus)
			return
		}
	})

	details.OnHTML("body", func(h *colly.HTMLElement) {
		names := h.ChildTexts("a.black.noline[data-productid]")
		if len(names) == 0 {
			return // not the right main table
		}
		publisher := ""
		publishDate := ""
		releaseYear := ""

		dataLinks := h.ChildAttrs("span.subheading > a", "href")
		dataTexts := h.ChildTexts("span.subheading > a")
		for i := 0; i < len(dataLinks); i++ {
			link := dataLinks[i]
			text := dataTexts[i]

			if strings.Contains(link, "movies.php?studioid") {
				publisher = text
			}

			if strings.Contains(link, "movies.php?year") {
				releaseYear = text
			}

			if strings.Contains(link, "releasedates.php") {
				publishDate = text
			}
		}
		runtime := h.ChildText("#runtime")

		msgChannel <- detailsMessage{
			Name:        names[0],
			Publisher:   publisher,
			PublishDate: publishDate,
			Runtime:     runtime,
			ReleaseYear: releaseYear,
		}

		msgChannel <- detailsMessage{
			EndSignal: true,
		}
	})

	mergingDatasetCount := 1 // details main table
	// timeout
	go func() {
		<-ctx.Done()
		for i := 0; i < mergingDatasetCount; i++ {
			msgChannel <- detailsMessage{EndSignal: true}
		}
	}()

	go func() {
		details.Visit(detailsLink)
	}()

	var msgs []detailsMessage
	endSignals := 0
	for {
		select {
		// todo: accumulate errors instead of breaking on first
		case err := <-errorChannel:
			return nil, err
		case msg := <-msgChannel:
			if msg.EndSignal {
				endSignals++
			}
			msgs = append(msgs, msg)
			if endSignals >= mergingDatasetCount {
				if out, ok := transformDetails(reduceDetails(msgs)); ok {
					return out, nil
				} else {
					return nil, ErrInsufficientDetails
				}
			}
		}
	}
}

func FullSearch(ctx context.Context, searchTerm string) ([]Bluray, error) {
	links, err := Search(ctx, searchTerm)
	if err != nil {
		return nil, err
	}

	out := make([]Bluray, 0, len(links))
	for _, link := range links {
		detail, err := GetDetails(ctx, link.BlurayDotComDetailsLink)
		if err != nil {
			return nil, err
		}
		out = append(out, Bluray{
			BlurayDetails: *detail,
			BlurayLinks:   link,
		})
	}

	return out, nil
}
