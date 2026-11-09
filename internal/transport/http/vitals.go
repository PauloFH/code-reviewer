package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// vitalsReport is one page load's Web Vitals, sent by the dashboard when the tab is hidden.
// Times are in milliseconds; CLS is unitless. A nil field means the browser did not measure it.
type vitalsReport struct {
	Path string   `json:"path"`
	TTFB *float64 `json:"ttfb"`
	FCP  *float64 `json:"fcp"`
	LCP  *float64 `json:"lcp"`
	CLS  *float64 `json:"cls"`
	INP  *float64 `json:"inp"`
}

// vitalsHandler records real-user Web Vitals as one structured log line per page load
// (`journalctl -u code-reviewer | grep "web vital"`), so no new storage is needed.
func vitalsHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v vitalsReport
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10)).Decode(&v); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(v.Path, "/") || len(v.Path) > 200 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		attrs := []any{slog.String("page", v.Path)}
		if claims, ok := userClaimsFromCtx(r); ok {
			attrs = append(attrs, slog.String("user", claims.Username))
		}
		for _, m := range []struct {
			name string
			val  *float64
		}{{"ttfb_ms", v.TTFB}, {"fcp_ms", v.FCP}, {"lcp_ms", v.LCP}, {"cls", v.CLS}, {"inp_ms", v.INP}} {
			if m.val == nil {
				continue
			}
			// Untrusted client input: an hour is far beyond any real measurement.
			if *m.val < 0 || *m.val > 3_600_000 {
				http.Error(w, "invalid "+m.name, http.StatusBadRequest)
				return
			}
			attrs = append(attrs, slog.Float64(m.name, *m.val))
		}

		log.Info("web vital", attrs...)
		w.WriteHeader(http.StatusNoContent)
	}
}
