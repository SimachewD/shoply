package response

type Response struct {
	Success bool `json:"success"`
	Message string `json:"message,omitempty"`
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
	Error any `json:"error,omitempty"`
}