package authtrace

import "time"

// timeNow is a thin indirection over time.Now so the classification logic can
// be unit-tested without depending on wall-clock time. Keep it package-local.
var timeNow = func() time.Time { return time.Now() }
