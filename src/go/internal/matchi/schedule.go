package matchi

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var (
	// slotTimePattern hittar start- och sluttid i en slots title-attribut.
	slotTimePattern = regexp.MustCompile(`(\d{1,2}):(\d{2})\s*-\s*(\d{1,2}):(\d{2})`)

	// groupIDPattern hittar grupp-id:t i en slots klass.
	groupIDPattern = regexp.MustCompile(`group_(\d+)`)

	// bookingIDPattern hittar det numeriska boknings-id:t i slotens attribut.
	bookingIDPattern = regexp.MustCompile(`\d+`)

	// colorClasses är klasser som beskriver färg och inte status.
	colorClasses = map[string]bool{
		"green": true, "blue": true, "yellow": true, "red": true,
		"grey": true, "gray": true, "purple": true, "orange": true,
		"pink": true, "brown": true, "teal": true, "cyan": true,
		"light-green": true, "light-blue": true, "light-yellow": true,
	}
)

// maxConcurrentSchedules begränsar hur många schemasidor som hämtas samtidigt.
//
// Varje schema är en egen HTML-sida på flera hundra kilobyte, så parallell
// hämtning gör att en hel period går igenom på ungefär en fjärdedel av tiden.
const maxConcurrentSchedules = 4

// DailySchedule är resultatet av att läsa ett dygns schema.
type DailySchedule struct {
	// Date är dagen som schemat gäller.
	Date time.Time

	// Schedule är schemat, eller nil om dagen inte kunde läsas.
	Schedule *Schedule

	// Err är felet som uppstod när dagen inte kunde läsas.
	Err error
}

// ScheduleForDays läser schemat för flera dagar parallellt.
//
// Resultaten kommer i samma ordning som dagarna som skickades in. En dag
// som inte gick att läsa får Err satt och Schedule nil, så att en enskild
// trasig dag inte får resten av perioden att falla.
func (c *Client) ScheduleForDays(ctx context.Context, days []time.Time) []DailySchedule {
	results := make([]DailySchedule, len(days))

	semaphore := make(chan struct{}, maxConcurrentSchedules)
	var wg sync.WaitGroup

	for i, day := range days {
		wg.Add(1)

		go func(index int, day time.Time) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			schedule, err := c.Schedule(ctx, day)
			results[index] = DailySchedule{Date: day, Schedule: schedule, Err: err}
		}(i, day)
	}

	wg.Wait()

	return results
}

// Schedule returnerar dagens schema för en anläggning.
//
// date anges i lokal tid (Europe/Stockholm) och ska vara midnatt den dag
// som ska läsas.
func (c *Client) Schedule(ctx context.Context, date time.Time) (*Schedule, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	day := date.In(c.location)
	_, body, err := c.get(ctx, "/facility/booking/index", url.Values{"date": {day.Format("2006-01-02")}})
	if err != nil {
		return nil, err
	}

	slots, err := parseSchedule(body, day, c.location)
	if err != nil {
		return nil, err
	}

	return &Schedule{Slots: slots}, nil
}

// parseSchedule läser ut slotterna ur MATCHis schemasida.
func parseSchedule(body string, day time.Time, loc *time.Location) ([]Slot, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("matchi: parse schema: %w", err)
	}

	var slots []Slot

	for _, td := range slotCells(doc) {
		slotID := attr(td, "slotid")
		if slotID == "" {
			continue
		}

		class := attr(td, "class")
		title := attr(td, "title")

		start, end, ok := parseSlotTimes(title, day, loc)
		if !ok {
			continue
		}

		court := courtName(td)

		slot := Slot{
			CourtName: court,
			SlotID:    slotID,
			Start:     start,
			End:       end,
			Free:      strings.Contains(class, "free"),
			Bookable:  !strings.Contains(class, "not-available"),
			Status:    parseStatus(class),
			BookingID: attr(td, "bookingid"),
			GroupID:   firstGroup(groupIDPattern, class),
			Title:     cleanTitle(title),
		}

		slots = append(slots, slot)
	}

	if len(slots) == 0 {
		return nil, fmt.Errorf("matchi: inga slotter hittades i schemat")
	}

	return slots, nil
}

// slotCells hittar alla tabellceller som representerar en tidsslot.
func slotCells(doc *html.Node) []*html.Node {
	var cells []*html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" && attr(n, "slotid") != "" {
			cells = append(cells, n)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return cells
}

// courtName hittar namnet på banan/salen som en slot tillhör.
func courtName(cell *html.Node) string {
	for n := cell; n != nil; n = n.Parent {
		if n.Type != html.ElementNode || n.Data != "tr" {
			continue
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode || child.Data != "td" {
				continue
			}
			if !hasClass(child, "court") {
				continue
			}

			if label := findByClass(child, "label"); label != nil {
				return strings.TrimSpace(textContent(label))
			}

			return strings.TrimSpace(textContent(child))
		}
	}

	return ""
}

// findByClass hittar första elementet med given klass.
func findByClass(n *html.Node, class string) *html.Node {
	var found *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}
		if node.Type == html.ElementNode && hasClass(node, class) {
			found = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)

	return found
}

// parseSlotTimes läser start- och sluttid ur en slots title-attribut.
func parseSlotTimes(title string, day time.Time, loc *time.Location) (time.Time, time.Time, bool) {
	matches := slotTimePattern.FindStringSubmatch(title)
	if matches == nil {
		return time.Time{}, time.Time{}, false
	}

	startHour, _ := strconv.Atoi(matches[1])
	startMinute, _ := strconv.Atoi(matches[2])
	endHour, _ := strconv.Atoi(matches[3])
	endMinute, _ := strconv.Atoi(matches[4])

	start := time.Date(day.Year(), day.Month(), day.Day(), startHour, startMinute, 0, 0, loc)
	end := time.Date(day.Year(), day.Month(), day.Day(), endHour, endMinute, 0, 0, loc)

	if !end.After(start) {
		// Bokning som går över midnatt.
		end = end.AddDate(0, 0, 1)
	}

	return start, end, true
}

// parseStatus plockar ut bokningsstatus ur en slots klass.
func parseStatus(class string) string {
	for _, token := range strings.Fields(class) {
		switch token {
		case "slot", "free", "not-available", "online":
			continue
		}
		if strings.HasPrefix(token, "group_") {
			continue
		}
		if colorClasses[token] {
			continue
		}
		return token
	}

	return ""
}

// cleanTitle tar bort HTML och onödig whitespace från en title-attribute.
func cleanTitle(title string) string {
	replacer := strings.NewReplacer("<br />", " ", "<br>", " ", "<br/>", " ", "<hr/>", " - ", "<hr>", " - ")

	cleaned := replacer.Replace(title)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return strings.TrimSpace(cleaned)
}

// attr läser en attribut från en nod.
func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}

	return ""
}

// hasClass avgör om noden har given klass.
func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}

	return false
}

// textContent läser texten i en nod.
func textContent(n *html.Node) string {
	var sb strings.Builder

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)

	return sb.String()
}

// firstGroup returnerar första gruppen i en matchning.
func firstGroup(pattern *regexp.Regexp, s string) string {
	matches := pattern.FindStringSubmatch(s)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
