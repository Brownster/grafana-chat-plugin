package plugin

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Message          string            `json:"message"`
	SessionID        string            `json:"session_id"`
	DashboardContext *DashboardContext `json:"dashboard_context,omitempty"`
}

// DashboardContext contains dashboard metadata
type DashboardContext struct {
	UID       string            `json:"uid"`
	Name      string            `json:"name"`
	Folder    string            `json:"folder"`
	Tags      []string          `json:"tags"`
	TimeRange map[string]string `json:"time_range"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}
