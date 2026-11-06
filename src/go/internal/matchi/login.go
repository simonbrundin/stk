package matchi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	// sessionCodePattern plockar ut session_code ur Keycloaks inloggningssida.
	//
	// Teckenklassen är generell eftersom Keycloaks sessions-id kan innehålla
	// tecken som "_" och ".", vilket skulle kapa värdet om man bara tillät
	// alfanumeriska tecken.
	sessionCodePattern = regexp.MustCompile(`session_code=([^&"'\s]+)`)

	// executionPattern plockar ut execution ur Keycloaks inloggningssida.
	executionPattern = regexp.MustCompile(`execution=([^&"'\s]+)`)

	// tabIDPattern plockar ut tab_id ur Keycloaks inloggningssida.
	tabIDPattern = regexp.MustCompile(`tab_id=([^&"'\s]+)`)

	// clientIDPattern plockar ut client_id ur Keycloaks inloggningssida.
	clientIDPattern = regexp.MustCompile(`client_id=([^&"'\s]+)`)
)

// loginAction bygger URL:en till Keycloaks inloggningsformulär utifrån
// inloggningssidan.
func loginAction(realmURL, body string) (string, error) {
	sessionCode := matchOrEmpty(sessionCodePattern, body)
	execution := matchOrEmpty(executionPattern, body)
	tabID := matchOrEmpty(tabIDPattern, body)

	if sessionCode == "" || execution == "" {
		return "", fmt.Errorf("matchi: kunde inte hitta Keycloak-parametrar i inloggningssidan")
	}

	clientID := matchOrEmpty(clientIDPattern, body)
	if clientID == "" {
		clientID = "webapp"
	}

	query := url.Values{
		"session_code": {sessionCode},
		"execution":    {execution},
		"client_id":    {clientID},
		"tab_id":       {tabID},
	}

	return realmURL + "/login-actions/authenticate?" + query.Encode(), nil
}

// matchOrEmpty returnerar första gruppen i en matchning, eller tom sträng.
func matchOrEmpty(pattern *regexp.Regexp, body string) string {
	m := pattern.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// readBody läser och stänger response-body.
func readBody(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// get utför en GET mot MATCHi och returnerar svaret som text.
func (c *Client) get(ctx context.Context, path string, query url.Values) (*http.Response, string, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", fmt.Errorf("matchi: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("matchi: GET %s: %w", path, err)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, "", fmt.Errorf("matchi: read %s: %w", path, err)
	}

	if c.isLoginPage(resp, body) {
		return nil, "", ErrUnauthenticated
	}

	return resp, body, nil
}

// postForm utför en POST med formulärdata mot MATCHi.
func (c *Client) postForm(ctx context.Context, path string, form url.Values) (*http.Response, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, "", fmt.Errorf("matchi: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("matchi: POST %s: %w", path, err)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, "", fmt.Errorf("matchi: read %s: %w", path, err)
	}

	if c.isLoginPage(resp, body) {
		return nil, "", ErrUnauthenticated
	}

	return resp, body, nil
}

// isLoginPage avgör om svaret är MATCHis inloggningssida.
func (c *Client) isLoginPage(resp *http.Response, body string) bool {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		if strings.HasPrefix(resp.Request.URL.Path, "/login") {
			return true
		}
	}

	return strings.Contains(body, "login-actions/authenticate")
}
