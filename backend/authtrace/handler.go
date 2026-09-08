package authtrace

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// ReportHandler is an UNAUTHENTICATED endpoint that lets the SvelteKit SSR
// layer and the browser record auth events into the same auth_trace table the
// backend writes to. It exists purely for debugging the logout problem.
//
// Body:
//
//	{
//	  "source": "ssr" | "browser",
//	  "event":  "ssr_redirect" | "client_redirect" | "heartbeat" | "client_observed",
//	  "path":   "/some/path",                 // the route the user was on
//	  "cookie_prefix": "ABC123" | "none",     // optional; server re-reads the cookie anyway
//	  "detail": { ... }                       // optional, arbitrary JSON
//	}
//
// The endpoint reads the session cookie itself (if present) so cookie_prefix
// is always trustworthy regardless of what the client claims. Source and
// event are constrained to a small allowlist so this can't be abused as an
// open log endpoint.
func ReportHandler(queries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Source       string          `json:"source"`
			Event        string          `json:"event"`
			Path         string          `json:"path"`
			CookiePrefix string          `json:"cookie_prefix"`
			Detail       json.RawMessage `json:"detail"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !allowedSource(body.Source) || !allowedEvent(body.Event) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source or event"})
			return
		}

		// Trust the server's view of the cookie over the client's claim, but
		// fall back to the client-reported prefix if the cookie wasn't sent on
		// this debug call (e.g. the cookie is already gone).
		token, _ := c.Cookie("session_token")
		if token == "" && body.CookiePrefix != "" && body.CookiePrefix != "none" {
			token = body.CookiePrefix
		}
		prefix := Prefix(token)

		// Surface the client-reported path inside detail so the row records
		// where the user actually was (this endpoint's own path is always
		// /api/_debug/auth-event, which isn't useful).
		var detail map[string]any
		if len(body.Detail) > 0 {
			_ = json.Unmarshal(body.Detail, &detail)
		}
		if detail == nil {
			detail = map[string]any{}
		}
		if body.Path != "" {
			detail["path"] = body.Path
		}

		raw, _ := json.Marshal(detail)
		LogRaw(queries, c, body.Source, body.Event, prefix, raw)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ViewHandler returns the most recent auth_trace rows. Admin-only (mounted on
// the admin-gated group) so it isn't exposed to regular users.
func ViewHandler(queries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := queries.RecentAuthTrace(c.Request.Context(), 200)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			row := map[string]any{
				"id":            r.ID,
				"created_at":    r.CreatedAt,
				"source":        r.Source,
				"event":         r.Event,
				"cookie_prefix": textStr(r.CookiePrefix),
				"ip":            textStr(r.IP),
				"method":        textStr(r.Method),
				"path":          textStr(r.Path),
			}
			if r.UserID.Valid {
				row["user_id"] = r.UserID.Int64
			}
			if len(r.Detail) > 0 {
				var d any
				if json.Unmarshal(r.Detail, &d) == nil {
					row["detail"] = d
				} else {
					row["detail"] = string(r.Detail)
				}
			}
			out = append(out, row)
		}
		c.JSON(http.StatusOK, gin.H{"events": out})
	}
}

func allowedSource(s string) bool {
	return s == "ssr" || s == "browser"
}

func allowedEvent(e string) bool {
	switch e {
	case string(ReasonSSRRedirect),
		string(ReasonSSRSetCookie),
		string(ReasonClientRedirect),
		string(ReasonHeartbeat),
		string(ReasonClientObserved):
		return true
	}
	return false
}

// CaptureSetCookieMiddleware logs every response that carries a Set-Cookie
// header mentioning session_token. This is the definitive way to find which
// backend response is mutating/deleting the session cookie — the auth_trace
// "random logout" investigation showed the cookie vanishing from the browser
// within seconds of login with no server-side logout, so we need to see every
// Set-Cookie the backend emits.
func CaptureSetCookieMiddleware(queries *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		values := c.Writer.Header().Values("Set-Cookie")
		if len(values) == 0 {
			return
		}
		for _, v := range values {
			if !strings.Contains(v, "session_token") {
				continue
			}
			token, _ := c.Cookie("session_token")
			category := "set"
			if strings.Contains(v, "Max-Age=0") || strings.Contains(v, "Max-Age=-1") ||
				strings.Contains(v, "session_token=;") {
				category = "delete"
			}
			Log(queries, c, "backend", "set_cookie", Prefix(token), map[string]any{
				"category": category,
				"header":   v,
			})
		}
	}
}

func textStr(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
