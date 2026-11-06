package matchi

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	// alertErrorPattern hittar felmeddelanden i MATCHis formulär.
	alertErrorPattern = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*alert-error[^"]*"[^>]*>(.*?)</div>`)

	// tagPattern hittar HTML-taggar i ett meddelande.
	tagPattern = regexp.MustCompile(`(?s)<[^>]*>`)
)

// parseFormFields läser ut formulärfälten ur en HTML-sida.
//
// Det används för att återanvända de dolda fält som MATCHis egna formulär
// sätter, till exempel versionsstämpeln på en bokning.
func parseFormFields(body string) (url.Values, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("matchi: parse formulär: %w", err)
	}

	values := url.Values{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "input":
				name := attr(n, "name")
				if name != "" && attr(n, "type") != "checkbox" && attr(n, "type") != "radio" {
					values.Set(name, attr(n, "value"))
				}
			case "textarea":
				name := attr(n, "name")
				if name != "" {
					values.Set(name, strings.TrimSpace(textContent(n)))
				}
			case "select":
				name := attr(n, "name")
				if name != "" {
					if value, ok := selectedOption(n); ok {
						values.Set(name, value)
					}
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return values, nil
}

// selectedOption returnerar värdet på det valda alternativet i en select.
func selectedOption(selectNode *html.Node) (string, bool) {
	var value string
	var found bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "option" {
			if hasAttr(n, "selected") {
				value = attr(n, "value")
				found = true
			}
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(selectNode)

	if !found {
		return "", false
	}

	return value, true
}

// hasAttr avgör om en nod har en given attribut.
func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if a.Key == name {
			return true
		}
	}

	return false
}

// errorMessage plockar ut ett felmeddelande ur ett svar.
func errorMessage(body string) string {
	matches := alertErrorPattern.FindStringSubmatch(body)
	if matches == nil {
		return ""
	}

	message := tagPattern.ReplaceAllString(matches[1], " ")
	message = strings.Join(strings.Fields(message), " ")

	return strings.TrimSpace(message)
}
