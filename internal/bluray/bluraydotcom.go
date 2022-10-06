package bluray

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/robertkrimen/otto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrOperationTimeout    = errors.New("operation timed out")
	ErrJavascriptEvaluate  = errors.New("failed to evaluate javascript")
	ErrJavascriptTranslate = errors.New("failed to translate javascript data")
	ErrBadResponseStatus   = errors.New("received bad response status")
	ErrInsufficientDetails = errors.New("couldn't find sufficient details")
	ErrMalformedData       = errors.New("found malformed data")
)

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
	Name                    string
	Publisher               string
	ReleaseYear             string
	Runtime                 string
	PublishDate             string
	BlurayDotComBuyID       string
	BlurayDotComDetailsLink string
	BlurayDotComImageLink   string
}

type searchMessage struct {
	EndSignal               bool
	Index                   int
	Display                 string
	BlurayDotComBuyID       string
	BlurayDotComDetailsLink string
	BlurayDotComImageLink   string
}

func (sm searchMessage) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_type", "search").
		Str("display", sm.Display).
		Str("link", sm.BlurayDotComDetailsLink).
		Bool("end_signal", sm.EndSignal)
}

type detailsMessage struct {
	EndSignal   bool
	Link        string
	Name        string
	Publisher   string
	PublishDate string
	Runtime     string
	ReleaseYear string
}

func (dm detailsMessage) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_type", "details").
		Str("name", dm.Name).
		Str("link", dm.Link).
		Bool("end_signal", dm.EndSignal)
}

type fullSearchMessage struct {
	EndSignal               bool
	ExpectDetailCount       int
	Index                   *int
	SearchDisplay           string
	Name                    string
	Publisher               string
	PublishDate             string
	Runtime                 string
	ReleaseYear             string
	BlurayDotComBuyID       string
	BlurayDotComDetailsLink string
	BlurayDotComImageLink   string
	Type                    string
}

func (fsm fullSearchMessage) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message_type", "full_search").
		Str("name", fsm.Name).
		Str("link", fsm.BlurayDotComDetailsLink).
		Bool("end_signal", fsm.EndSignal)
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
	if a.BlurayDotComBuyID != "" {
		out.BlurayDotComBuyID = a.BlurayDotComBuyID
	} else if b.BlurayDotComBuyID != "" {
		out.BlurayDotComBuyID = b.BlurayDotComBuyID
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
		BlurayDotComBuyID:       msg.BlurayDotComBuyID,
		BlurayDotComDetailsLink: msg.BlurayDotComDetailsLink,
		BlurayDotComImageLink:   msg.BlurayDotComImageLink,
	}

	return out, out.ShortName != "" || out.BlurayDotComBuyID != "" || out.BlurayDotComDetailsLink != ""
}

func searchToFull(msg searchMessage) fullSearchMessage {
	t := "search"
	if msg.EndSignal {
		t = "end"
	}
	return fullSearchMessage{
		EndSignal:               msg.EndSignal,
		Index:                   &msg.Index,
		SearchDisplay:           msg.Display,
		BlurayDotComBuyID:       msg.BlurayDotComBuyID,
		BlurayDotComDetailsLink: msg.BlurayDotComDetailsLink,
		BlurayDotComImageLink:   msg.BlurayDotComImageLink,
		Type:                    t,
	}
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

func reduceDetails(msgs []detailsMessage) detailsMessage {
	out := detailsMessage{}

	for _, msg := range msgs {
		if msg.EndSignal {
			out.EndSignal = true
		}
		if msg.Link != "" {
			out.Link = msg.Link
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

func detailsToFull(msg detailsMessage) fullSearchMessage {
	t := "details"
	if msg.EndSignal {
		t = "end"
	}
	return fullSearchMessage{
		EndSignal:               msg.EndSignal,
		BlurayDotComDetailsLink: msg.Link,
		Name:                    msg.Name,
		Publisher:               msg.Publisher,
		PublishDate:             msg.PublishDate,
		Runtime:                 msg.Runtime,
		ReleaseYear:             msg.ReleaseYear,
		Type:                    t,
	}
}

func mergeFullSearch(a, b fullSearchMessage) fullSearchMessage {
	out := fullSearchMessage{}
	if a.EndSignal || b.EndSignal {
		out.EndSignal = true
	}
	if a.Index != nil {
		out.Index = a.Index
	} else if b.Index != nil {
		out.Index = b.Index
	}
	if a.SearchDisplay != "" {
		out.SearchDisplay = a.SearchDisplay
	} else if b.SearchDisplay != "" {
		out.SearchDisplay = b.SearchDisplay
	}
	if a.Name != "" {
		out.Name = a.Name
	} else if b.Name != "" {
		out.Name = b.Name
	}
	if a.Publisher != "" {
		out.Publisher = a.Publisher
	} else if b.Publisher != "" {
		out.Publisher = b.Publisher
	}
	if a.PublishDate != "" {
		out.PublishDate = a.PublishDate
	} else if b.PublishDate != "" {
		out.PublishDate = b.PublishDate
	}
	if a.Runtime != "" {
		out.Runtime = a.Runtime
	} else if b.Runtime != "" {
		out.Runtime = b.Runtime
	}
	if a.ReleaseYear != "" {
		out.ReleaseYear = a.ReleaseYear
	} else if b.ReleaseYear != "" {
		out.ReleaseYear = b.ReleaseYear
	}
	if a.BlurayDotComBuyID != "" {
		out.BlurayDotComBuyID = a.BlurayDotComBuyID
	} else if b.BlurayDotComBuyID != "" {
		out.BlurayDotComBuyID = b.BlurayDotComBuyID
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

func collectFullSearch(msgs []fullSearchMessage) []fullSearchMessage {
	byIndex := make(map[int][]fullSearchMessage, 0)
	byLink := make(map[string][]fullSearchMessage, 0)
	var out []fullSearchMessage

	for _, msg := range msgs {
		if msg.Index != nil {
			byIndex[*msg.Index] = append(byIndex[*msg.Index], msg)
		}
		if msg.BlurayDotComDetailsLink != "" {
			byLink[msg.BlurayDotComDetailsLink] = append(byLink[msg.BlurayDotComDetailsLink], msg)
		}
	}

	for _, msgs := range byLink {
		workingSet := []fullSearchMessage{}
		workingSet = append(workingSet, msgs...)
		for _, msg := range msgs {
			if msg.Index != nil {
				workingSet = append(workingSet, byIndex[*msg.Index]...)
			}
		}
		merged := fullSearchMessage{}
		for _, msg := range workingSet {
			merged = mergeFullSearch(merged, msg)
		}
		out = append(out, merged)
	}

	return out
}

func transformFullSearch(msg fullSearchMessage) (Bluray, bool) {
	out := Bluray{
		Name:                    msg.Name,
		Publisher:               msg.Publisher,
		ReleaseYear:             msg.ReleaseYear,
		Runtime:                 msg.Runtime,
		PublishDate:             msg.PublishDate,
		BlurayDotComBuyID:       msg.BlurayDotComBuyID,
		BlurayDotComDetailsLink: msg.BlurayDotComDetailsLink,
		BlurayDotComImageLink:   msg.BlurayDotComImageLink,
	}

	fallbackName, fallbackPublishDate := normalizeSearchDisplay(msg.SearchDisplay)
	if out.Name == "" {
		out.Name = fallbackName
	}
	if out.PublishDate == "" {
		out.PublishDate = fallbackPublishDate
	}

	return out, out.Name != "" || out.BlurayDotComDetailsLink != ""
}

func transformAllFullSearch(msgs []fullSearchMessage) []Bluray {
	out := make([]Bluray, 0, len(msgs))
	for _, msg := range msgs {
		if blu, ok := transformFullSearch(msg); ok {
			out = append(out, blu)
		}
	}
	return out
}

func quicksearchMiddleware() func(*colly.Request) {
	return func(r *colly.Request) {
		r.Headers.Set("X-Requested-With", "XMLHttpRequest")
		r.Headers.Set("Referer", "https://www.blu-ray.com")
		r.Headers.Set("Cookie", "pw_bottom_filter=blur; firstview=1; search_section=bluraymovies")
	}
}

func detailsMiddleware() func(*colly.Request) {
	return func(r *colly.Request) {
		r.Headers.Set("Referer", "https://www.blu-ray.com")
		r.Headers.Set("Cookie", "pw_bottom_filter=blur; firstview=1; search_section=bluraymovies")
	}
}

func parseQuicksearchJS(js *otto.Otto, errorChannel chan<- error, msgChannel chan<- searchMessage) func(*colly.HTMLElement) {
	return func(h *colly.HTMLElement) {
		for _, jsLine := range strings.Split(h.Text, ";") {
			trimmed := strings.TrimSpace(jsLine)
			termed := fmt.Sprintf("%s;", trimmed)
			if strings.HasPrefix(termed, "var") || strings.HasPrefix(termed, "const") {
				_, err := js.Run(termed)
				if err != nil {
					errorChannel <- fmt.Errorf("running js from blu-ray.com quicksearch: %w", ErrJavascriptEvaluate)
					return
				}
			}
		}

		ids := evalJSArray(js, errorChannel, "ids")
		urls := evalJSArray(js, errorChannel, "urls")
		images := evalJSArray(js, errorChannel, "images")
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
				BlurayDotComBuyID:       id,
				BlurayDotComDetailsLink: url,
				BlurayDotComImageLink:   image,
			}
			i++
		}

		msgChannel <- searchMessage{
			EndSignal: true,
			Index:     i,
		}
	}
}

func parseQuicksearchList(errorChannel chan<- error, msgChannel chan<- searchMessage) func(*colly.HTMLElement) {
	return func(h *colly.HTMLElement) {
		lastIndex := 0
		for i, text := range h.ChildTexts("li") {
			msgChannel <- searchMessage{
				Display: text,
				Index:   i,
			}
			lastIndex = i
		}
		msgChannel <- searchMessage{
			EndSignal: true,
			Index:     lastIndex + 1,
		}
	}
}

func parseDetailsBody(errorChannel chan<- error, msgChannel chan<- detailsMessage) func(*colly.HTMLElement) {
	return func(h *colly.HTMLElement) {
		names := h.ChildTexts("a.black.noline[data-productid]")
		if len(names) == 0 {
			names = h.ChildTexts("h1")
		}
		if len(names) == 0 {
			errorChannel <- fmt.Errorf("details from url(%s): %w", h.Request.URL.String(), ErrMalformedData)
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
			Link:        h.Request.URL.String(),
			Name:        names[0],
			Publisher:   publisher,
			PublishDate: publishDate,
			Runtime:     runtime,
			ReleaseYear: releaseYear,
		}

		msgChannel <- detailsMessage{
			EndSignal: true,
		}
	}
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

func reportBadResponses(errorChannel chan<- error) func(r *colly.Response) {
	return func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			errorChannel <- fmt.Errorf("got status code %d from blu-ray.com quicksearch: %w", r.StatusCode, ErrBadResponseStatus)
		}
	}
}

// todo: really brittle js evaluation, figure out a better way
func evalJSArray(js *otto.Otto, errorChannel chan<- error, varName string) []string {
	var out []string

	jsVar, err := js.Get(varName)
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
		return nil
	}

	return out
}

func Search(ctx context.Context, searchTerm string) ([]BlurayLinks, error) {
	quicksearch := colly.NewCollector()
	quicksearchJS := otto.New()
	errorChannel := make(chan error)
	msgChannel := make(chan searchMessage)

	quicksearch.OnRequest(quicksearchMiddleware())
	quicksearch.OnResponse(reportBadResponses(errorChannel))
	quicksearch.OnHTML("script", parseQuicksearchJS(quicksearchJS, errorChannel, msgChannel))
	quicksearch.OnHTML("ul", parseQuicksearchList(errorChannel, msgChannel))

	widestDataBreadth := 2 // search hit links, search hit display text
	// timeout
	go func() {
		<-ctx.Done()
		errorChannel <- ErrOperationTimeout
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
			if endSignals >= widestDataBreadth {
				return transformAllSearch(collectSearch(msgs)), nil
			}
		}
	}
}

func GetDetails(ctx context.Context, detailsLink string) (*BlurayDetails, error) {
	details := colly.NewCollector()
	errorChannel := make(chan error)
	msgChannel := make(chan detailsMessage)

	details.OnRequest(detailsMiddleware())
	details.OnResponse(reportBadResponses(errorChannel))
	details.OnHTML("body", parseDetailsBody(errorChannel, msgChannel))

	widestDataBreadth := 1 // details main table
	// timeout
	go func() {
		<-ctx.Done()
		errorChannel <- ErrOperationTimeout
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
			if endSignals >= widestDataBreadth {
				if out, ok := transformDetails(reduceDetails(msgs)); ok {
					return out, nil
				} else {
					return nil, ErrInsufficientDetails
				}
			}
		}
	}
}

var (
	FullSearchMinParallelism = 1
	FullSearchMaxParallelism = 20
)

type collyLogging struct{}

func (d *collyLogging) Init() error {
	return nil
}

func (d *collyLogging) Event(e *debug.Event) {
	log.Debug().
		Str("colly_event", e.Type).
		Fields(e.Values).
		Uint32("request_id", e.RequestID).
		Uint32("collector_id", e.CollectorID).
		Msg("colly")
}

func FullSearch(ctx context.Context, parallelism uint, searchTerm string) ([]Bluray, error) {
	maxDataBreadth := 1 // 2 // search hit links, search hit display text, since details are a single width child
	effectiveParallelism := int(parallelism)
	if effectiveParallelism == 0 {
		effectiveParallelism = FullSearchMaxParallelism
	} else if effectiveParallelism < FullSearchMinParallelism {
		effectiveParallelism = FullSearchMinParallelism
	} else if effectiveParallelism > FullSearchMaxParallelism {
		effectiveParallelism = FullSearchMaxParallelism
	}

	quicksearch := colly.NewCollector(colly.Async(true), colly.MaxDepth(1))
	quicksearch.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: effectiveParallelism})
	details := colly.NewCollector(colly.Async(true), colly.MaxDepth(1), colly.Debugger(&collyLogging{}))
	details.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: effectiveParallelism})
	// detailsRequestQueue, _ := queue.New(effectiveParallelism, &queue.InMemoryQueueStorage{MaxSize: 10000})
	quicksearchJS := otto.New()
	errorChannel := make(chan error)
	searchChannel := make(chan searchMessage)
	detailChannel := make(chan detailsMessage)
	msgChannel := make(chan fullSearchMessage)

	quicksearch.OnError(func(r *colly.Response, err error) { errorChannel <- err })
	details.OnError(func(r *colly.Response, err error) { errorChannel <- err })
	quicksearch.OnRequest(quicksearchMiddleware())
	details.OnRequest(detailsMiddleware())
	quicksearch.OnResponse(reportBadResponses(errorChannel))
	details.OnResponse(reportBadResponses(errorChannel))
	quicksearch.OnHTML("script", parseQuicksearchJS(quicksearchJS, errorChannel, searchChannel))
	// quicksearch.OnHTML("ul", parseQuicksearchList(errorChannel, searchChannel))
	details.OnHTML("body", parseDetailsBody(errorChannel, detailChannel))

	// timeout
	go func() {
		<-ctx.Done()
		err := ErrOperationTimeout
		log.Error().
			AnErr("err", err).
			Msg("timeout")
		errorChannel <- err
	}()

	// search
	go func() {
		request := make(map[string]string, 4)
		request["section"] = "bluraymovies"
		request["userid"] = "-1"
		request["country"] = "all"
		request["keyword"] = searchTerm
		quicksearch.Post("https://www.blu-ray.com/search/quicksearch.php", request)
	}()

	// enqueue details requests
	go func() {
		for message := range searchChannel {
			// hold the msg for concurrency-safe access
			msg := message
			log.Debug().
				EmbedObject(msg).
				Msg("received event")
			if msg.BlurayDotComDetailsLink != "" {
				// detailsRequestQueue.AddURL(msg.BlurayDotComDetailsLink)
				details.Visit(msg.BlurayDotComDetailsLink)
			}
			if msg.EndSignal {
				msgChannel <- fullSearchMessage{
					ExpectDetailCount: msg.Index,
				}
			}
			msgChannel <- searchToFull(msg)
		}
	}()

	// concat details back into working set
	go func() {
		for message := range detailChannel {
			// hold the msg for concurrency-safe access
			msg := message
			log.Debug().
				EmbedObject(msg).
				Msg("received event")
			msgChannel <- detailsToFull(msg)
		}
	}()

	var msgs []fullSearchMessage
	endSignals := 0
	detailCountKnown := false
	for {
		select {
		// todo: accumulate errors instead of breaking on first
		case err := <-errorChannel:
			return nil, err
		case msg := <-msgChannel:
			log.Debug().
				EmbedObject(msg).
				Msg("received event")
			if msg.ExpectDetailCount != 0 && !detailCountKnown {
				detailCountKnown = true
				endSignals -= msg.ExpectDetailCount
			}
			if msg.EndSignal {
				endSignals++
			}
			msgs = append(msgs, msg)
			if endSignals >= maxDataBreadth && detailCountKnown {
				return transformAllFullSearch(collectFullSearch(msgs)), nil
			}
		}
	}
}
