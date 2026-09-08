#!/usr/bin/env bash
# auth-trace.sh — print the recent auth_trace timeline to diagnose random logouts.
#
# Every auth/session event from three layers lands in auth_trace:
#   - backend  : RequireAuth 401s, login, logout, sliding renewal, session-check
#   - ssr      : SvelteKit hooks.server.ts redirects to /login (with the reason)
#   - browser  : client-side session heartbeat + observed 401s
#
# Usage:
#   ./scripts/auth-trace.sh            # last 40 events
#   ./scripts/auth-trace.sh 120        # last 120 events
#
# The most useful columns: created_at (timeline), source (which layer),
# event (what happened), cookie_prefix (correlate the same browser session
# across rows — "none" means the browser sent no cookie), and detail.reason
# (no_cookie | session_invalid_401 | network_error | expired | unknown_token
# | http_500 | no_session_cookie | ...).
set -euo pipefail

LIMIT="${1:-40}"
DB_CONTAINER="${DB_CONTAINER:-fgb-db1}"

docker exec "$DB_CONTAINER" psql -U postgres -d example_db -c \
  "select to_char(created_at at time zone 'UTC','HH24:MI:SS') as time,
          source,
          event,
          cookie_prefix as cookie,
          coalesce(detail->>'reason', detail->>'category','') as reason,
          coalesce(detail->>'path', path, '') as path,
          coalesce(detail->>'header','') as set_cookie_header,
          coalesce(detail->>'was','') || case when detail->>'status' is not null then '->'||(detail->>'status') else '' end as transition
   from auth_trace
   order by created_at desc
   limit ${LIMIT};"
