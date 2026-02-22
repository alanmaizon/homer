package domain

type TaskType string
type AgentRole string

const (
	TaskSummarize TaskType = "summarize"
	TaskRewrite   TaskType = "rewrite"

	RolePlanner  AgentRole = "planner"
	RoleExecutor AgentRole = "executor"
	RoleCritic   AgentRole = "critic"
)

type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type TaskRequest struct {
	Task         TaskType   `json:"task"`
	Documents    []Document `json:"documents"`
	Text         string     `json:"text"`
	Mode         string     `json:"mode"`
	Instructions string     `json:"instructions"`
	Style        string     `json:"style"`
	EnableCritic bool       `json:"enableCritic"`
}

type PlanStep struct {
	ID     string    `json:"id"`
	Role   AgentRole `json:"role"`
	Action string    `json:"action"`
}

type TaskResponse struct {
	Result   string     `json:"result"`
	Plan     []PlanStep `json:"plan"`
	Metadata Metadata   `json:"metadata"`
}

type Metadata struct {
	Provider        string `json:"provider"`
	ExecutionTimeMs int64  `json:"executionTimeMs"`
	RequestID       string `json:"requestId,omitempty"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

type CapabilitiesResponse struct {
	Runtime  RuntimeCapabilities `json:"runtime"`
	Features FeatureFlags        `json:"features"`
}

type RuntimeCapabilities struct {
	RequestedProvider  string `json:"requestedProvider"`
	ActiveProvider     string `json:"activeProvider"`
	ProviderFallback   bool   `json:"providerFallback"`
	RequestedConnector string `json:"requestedConnector"`
	ActiveConnector    string `json:"activeConnector"`
	ConnectorFallback  bool   `json:"connectorFallback"`
}

type FeatureFlags struct {
	Critic          bool `json:"critic"`
	ConnectorImport bool `json:"connectorImport"`
	ConnectorExport bool `json:"connectorExport"`
}

type ConnectorImportRequest struct {
	DocumentID string `json:"documentId"`
}

type ConnectorImportResponse struct {
	Connector string   `json:"connector"`
	Document  Document `json:"document"`
}

type ConnectorExportRequest struct {
	DocumentID string `json:"documentId"`
	Content    string `json:"content"`
}

type ConnectorExportResponse struct {
	Connector string `json:"connector"`
	Exported  bool   `json:"exported"`
}
