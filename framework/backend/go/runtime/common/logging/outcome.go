package logging

type OutcomeStatus string

const (
	OutcomeSuccess  OutcomeStatus = "success"
	OutcomeFailed   OutcomeStatus = "failed"
	OutcomeRetrying OutcomeStatus = "retrying"
	OutcomeDropped  OutcomeStatus = "dropped"
)

type SinkOutcome struct {
	Sink      SinkType      `json:"sink"`
	Status    OutcomeStatus `json:"status"`
	Attempt   int           `json:"attempt"`
	ErrorCode string        `json:"error_code,omitempty"`
	Error     string        `json:"error_message,omitempty"`
}
