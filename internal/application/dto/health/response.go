package healthdto

import "time"

type HealthResponse struct {
	Status    string    `json:"status"`
	App       string    `json:"app"`
	Database  string    `json:"database"`
	Timestamp time.Time `json:"timestamp"`
}
