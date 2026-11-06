package matchi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client pratar med MATCHis adminapplikation via cookie-baserad inloggning.
//
// Klienten håller en cookie-jar och loggar in vid behov. Inloggningen
// serialiseras med en mutex eftersom flera anrop kan komma samtidigt.
type Client struct {
	baseURL      string
	authRealmURL string
	username     string
	password     string
	location     *time.Location
	httpClient   *http.Client

	mu        sync.Mutex
	loggedIn  bool
	loginErr  error
	loginTime time.Time
}

// NewClient skapar en klient mot MATCHi.
//
// Tomma argument faller tillbaka på defaultvärdena: bas-URL, realm, timeout
// och tidszon.
func NewClient(baseURL, username, password string, timeout time.Duration, location *time.Location) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if location == nil {
		location, _ = time.LoadLocation(DefaultLocation)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		jar = nil
	}

	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		authRealmURL: DefaultAuthRealmPath,
		username:     username,
		password:     password,
		location:     location,
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
	}
}

// isEncryptedSecret identifierar ett värde som inte dekrypterats.
//
// SOPS krypterade värden börjar med "ENC[" - om ett sådant värde skickas
// som lösenord till Keycloak räknas det som ett ogiltigt inloggningsförsök.
func isEncryptedSecret(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "ENC[")
}

// Location returnerar tidszonen klienten tolkar tiderna i.
func (c *Client) Location() *time.Location {
	return c.location
}

// Login loggar in i MATCHi och sparar sessionens cookies.
//
// Inloggningen görs om varje gång sessionen har gått ut, vilket händer
// efter ett antal timmar.
func (c *Client) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.login(ctx)
}

// login utför inloggningen. Anropas med låsen held.
func (c *Client) login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		return errors.New("matchi: användarnamn och lösenord krävs")
	}

	// Krypterade hemligheter får aldrig skickas som lösenord. Ett felaktigt
	// dekrypterat secret skulle annars ge Keycloak ogiltiga inloggningsförsök
	// och riskera att kontot låses av brute force-skyddet.
	if isEncryptedSecret(c.password) {
		return errors.New("matchi: lösenordet ser ut att vara krypterat - kontrollera att secreten är dekrypterad")
	}

	// Steg 1: Starta inloggningen. Kedjan av omdirigeringar slutar i
	// Keycloak, där vi får session_code, execution och tab_id.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/login", nil)
	if err != nil {
		return fmt.Errorf("matchi: create login request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("matchi: start login: %w", err)
	}
	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return fmt.Errorf("matchi: read login page: %w", err)
	}

	realmURL := c.realmURLFrom(resp.Request.URL)
	action, err := loginAction(realmURL, body)
	if err != nil {
		return err
	}

	// Steg 2: Skicka användarnamn och lösenord till Keycloak.
	form := url.Values{
		"username":     {c.username},
		"password":     {c.password},
		"credentialId": {""},
	}

	authReq, err := http.NewRequestWithContext(ctx, http.MethodPost, action, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("matchi: create auth request: %w", err)
	}
	authReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	authResp, err := c.httpClient.Do(authReq)
	if err != nil {
		return fmt.Errorf("matchi: authenticate: %w", err)
	}
	defer authResp.Body.Close()

	if _, err := readBody(authResp); err != nil {
		return fmt.Errorf("matchi: read auth response: %w", err)
	}

	// Steg 3: Kontrollera att vi landade på MATCHi och inte på login-sidan.
	finalURL := authResp.Request.URL
	if finalURL.Host != "" && !strings.Contains(finalURL.Host, hostOf(c.baseURL)) {
		return fmt.Errorf("matchi: inloggning misslyckades, hamnade på %s", finalURL.Host)
	}
	if strings.HasPrefix(finalURL.Path, "/login") {
		return errors.New("matchi: inloggning misslyckades, sessionen återfördes till inloggningen")
	}

	c.loggedIn = true
	c.loginTime = time.Now()

	return nil
}

// ensureLogin loggar in om sessionen saknas eller är för gammal.
func (c *Client) ensureLogin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loggedIn && time.Since(c.loginTime) < sessionTTL {
		return nil
	}

	return c.login(ctx)
}

// sessionTTL är hur länge en session antas hålla utan att förnya den.
const sessionTTL = 30 * time.Minute

// realmURLFrom räknar ut Keycloak-realms URL utifrån den URL som
// inloggningen slutade på.
func (c *Client) realmURLFrom(u *url.URL) string {
	if u == nil {
		return c.baseURL + c.authRealmURL
	}

	if idx := strings.Index(u.Path, "/protocol/"); idx > 0 {
		return u.Scheme + "://" + u.Host + u.Path[:idx]
	}

	return c.baseURL + c.authRealmURL
}

// hostOf returnerar värden på en URL.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Host
}
