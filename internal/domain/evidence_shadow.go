package domain

import "time"

type EvidenceShadowMetrics struct {
	WindowStart              time.Time `json:"windowStart"`
	WindowDays               int       `json:"windowDays"`
	AndroidObserved          int       `json:"androidObserved"`
	AndroidParseable         int       `json:"androidParseable"`
	AndroidComplete          int       `json:"androidComplete"`
	LibGMObserved            int       `json:"libgmObserved"`
	LibGMComplete            int       `json:"libgmComplete"`
	ExactMatches             int       `json:"exactMatches"`
	AndroidOnlyComplete      int       `json:"androidOnlyComplete"`
	LibGMOnlyComplete        int       `json:"libgmOnlyComplete"`
	ReferenceCoveragePercent float64   `json:"referenceCoveragePercent"`
	ExactParityPercent       float64   `json:"exactParityPercent"`
	RemovalReady             bool      `json:"removalReady"`
	RemovalGate              string    `json:"removalGate"`
}
