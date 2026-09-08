// Package authtrace is a dedicated, unified log for authentication/session
// events used to diagnose the "randomly logged out" problem. Events are
// written from three places:
//
//   - the backend (RequireAuth 401s, login, logout, sliding renewal),
//   - the SvelteKit SSR layer (hooks.server.ts redirect reasons), and
//   - the browser (a client-side session heartbeat + observed 401s).
//
// Everything lands in the auth_trace table so there is ONE timestamped,
// queryable timeline showing which layer dropped the session and why.
//
// All logging here is best-effort: an insert failure must never fail the
// request being traced.
package authtrace

import (
	"context"
	"encoding/json"
	database "fgb-lp/database/queries"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// Reason is the outcome of examining a request's session cookie + DB lookup.
// It is the single most useful field for diagnosing a logout: it tells you
// whether the browser even sent a cookie, and if so whether the server knows
// about it.
type Reason string

const (
	ReasonNoCookie       Reason = "no_cookie"       // browser sent no session_token cookie at all
	ReasonUnknownToken   Reason = "unknown_token"   // cookie present but no matching session row
	ReasonExpired        Reason = "expired"         // session row exists but expires_at <= now()
	ReasonDBError        Reason = "db_error"        // session lookup failed with a DB error
	ReasonInvalidUser    Reason = "invalid_user_id" // session valid but user lookup failed
	ReasonLoginCreated   Reason = "login_created"   // a fresh session was just created
	ReasonSessionTouched Reason = "session_touched" // sliding renewal extended the session
	ReasonLogout         Reason = "logout"          // explicit logout
	ReasonSSRRedirect    Reason = "ssr_redirect"    // SvelteKit SSR redirected to /login
	ReasonSSRSetCookie   Reason = "ssr_set_cookie"  // SvelteKit SSR set/deleted the session cookie
	ReasonClientRedirect Reason = "client_redirect" // SvelteKit client 401 redirected to /login
	ReasonHeartbeat      Reason = "heartbeat"       // periodic client-side session check
	ReasonClientObserved Reason = "client_observed" // client saw some auth-relevant event
)

// Prefix returns a short, non-reversible prefix of the session cookie so the
// same browser session can be correlated across rows. The full token is the
// only credential, so we never persist more than the first 6 chars (30 bits
// out of 120 — far too little to forge). "none" means no cookie was sent.
func Prefix(token string) string {
	if token == "" {
		return "none"
	}
	if len(token) <= 6 {
		return token
	}
	return token[:6]
}

// Classify explains why an authenticated request failed. It does one extra
// cheap DB lookup (only on the failure path) to separate "expired" from
// "unknown token", which GetSessionByToken collapses into a single error.
func Classify(ctx context.Context, queries *database.Queries, token string) Reason {
	if strings.TrimSpace(token) == "" {
		return ReasonNoCookie
	}
	row, err := queries.GetSessionByTokenAnyExpiry(ctx, token)
	if err != nil {
		return ReasonUnknownToken
	}
	if !row.ExpiresAt.Valid || row.ExpiresAt.Time.IsZero() {
		return ReasonExpired
	}
	if !row.ExpiresAt.Time.After(timeNow()) {
		return ReasonExpired
	}
	return ReasonDBError // row present + not expired → shouldn't happen, flag it
}

// Log writes one auth_trace row. Fire-and-forget.
//   - c may be nil (background / non-HTTP callers).
//   - cookiePrefix is the result of Prefix(token); pass "none" if unknown.
//   - detail is a free-form map serialized to JSONB.
func Log(queries *database.Queries, c *gin.Context, source, event, cookiePrefix string, detail map[string]any) {
	params := buildParams(c, source, event, cookiePrefix, detail)
	if params == nil {
		return
	}
	if err := queries.InsertAuthTrace(context.Background(), *params); err != nil {
		log.Printf("[authtrace] insert failed (event=%s): %v", event, err)
	}
}

// LogRaw is like Log but takes a pre-serialized detail blob. Used by the
// public debug endpoint that already receives JSON from the frontend/browser.
func LogRaw(queries *database.Queries, c *gin.Context, source, event, cookiePrefix string, detail json.RawMessage) {
	detailBytes := []byte(detail)
	if len(detailBytes) == 0 {
		detailBytes = []byte("{}")
	}
	params := &database.InsertAuthTraceParams{
		Source:       source,
		Event:        event,
		IP:           clientIP(c),
		Method:       method(c),
		Path:         path(c),
		CookiePrefix: pgtype.Text{String: cookiePrefix, Valid: true},
		Detail:       detailBytes,
	}
	if err := queries.InsertAuthTrace(context.Background(), *params); err != nil {
		log.Printf("[authtrace] insert failed (event=%s): %v", event, err)
	}
}

func buildParams(c *gin.Context, source, event, cookiePrefix string, detail map[string]any) *database.InsertAuthTraceParams {
	detailBytes, err := json.Marshal(detail)
	if err != nil {
		log.Printf("[authtrace] marshal detail failed (event=%s): %v", event, err)
		detailBytes = []byte("{}")
	}
	return &database.InsertAuthTraceParams{
		Source:       source,
		Event:        event,
		UserID:       userID(c),
		IP:           clientIP(c),
		Method:       method(c),
		Path:         path(c),
		CookiePrefix: pgtype.Text{String: cookiePrefix, Valid: true},
		Detail:       detailBytes,
	}
}

func userID(c *gin.Context) pgtype.Int8 {
	if c == nil {
		return pgtype.Int8{Valid: false}
	}
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return pgtype.Int8{Int64: id, Valid: true}
		}
	}
	return pgtype.Int8{Valid: false}
}

func clientIP(c *gin.Context) pgtype.Text {
	if c == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: c.ClientIP(), Valid: true}
}

func method(c *gin.Context) pgtype.Text {
	if c == nil || c.Request == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: c.Request.Method, Valid: true}
}

func path(c *gin.Context) pgtype.Text {
	if c == nil || c.Request == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: c.Request.URL.RequestURI(), Valid: true}
}
