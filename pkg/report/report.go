package report

import (
	"time"
)

type Handler interface {
	Handle(report *Report) error
}

type Report struct {
	Status   RunStatus      `json:"status,omitempty"`
	Metric   *FuzzingMetric `json:"metric,omitempty"`
	Finding  *Finding       `json:"finding,omitempty"`
	NumSeeds uint           `json:"num_seeds,omitempty"`
}

func (x *Report) GetFinding() *Finding {
	if x != nil {
		return x.Finding
	}
	return nil
}

type RunStatus string

// These constants must have this exact value (in uppercase) to be able
// to parse JSON-marshalled reports as protobuf reports which use an
// enum for this field.
const (
	RunStatus_RUNSTATUS_UNSPECIFIED      RunStatus = "UNSPECIFIED"
	RunStatus_PENDING                    RunStatus = "PENDING"
	RunStatus_COMPILING                  RunStatus = "COMPILING"
	RunStatus_RUNNING                    RunStatus = "RUNNING"
	RunStatus_STOPPED                    RunStatus = "STOPPED"
	RunStatus_FAILED                     RunStatus = "FAILED"
	RunStatus_SUCCEEDED                  RunStatus = "SUCCEEDED"
	RunStatus_UNKNOWN                    RunStatus = "UNKNOWN"
	RunStatus_INITIALIZING               RunStatus = "INITIALIZING"
	RunStatus_FAILED_TO_START            RunStatus = "FAILED_TO_START"
	RunStatus_WAITING_FOR_FUZZING_AGENTS RunStatus = "WAITING_FOR_FUZZING_AGENTS"
)

type FuzzingMetric struct {
	Timestamp               time.Time `json:"timestamp,omitempty"`
	ExecutionsPerSecond     int32     `json:"executions_per_second,omitempty"`
	Features                int32     `json:"features,omitempty"`
	CorpusSize              int32     `json:"corpus_size,omitempty"`
	SecondsSinceLastFeature uint64    `json:"seconds_since_last_coverage,omitempty"`
	TotalExecutions         uint64    `json:"total_executions,omitempty"`
	Edges                   int32     `json:"edges,omitempty"`
	SecondsSinceLastEdge    uint64    `json:"seconds_since_last_edge,omitempty"`
}

type ErrorType string

// These constants must have this exact value (in uppercase) to be able
// to parse JSON-marshalled reports as protobuf reports which use an
// enum for this field.
const (
	ErrorType_UNKNOWN_ERROR     ErrorType = "UNKNOWN_ERROR"
	ErrorType_COMPILATION_ERROR ErrorType = "COMPILATION_ERROR"
	ErrorType_CRASH             ErrorType = "CRASH"
	ErrorType_WARNING           ErrorType = "WARNING"
	ErrorType_RUNTIME_ERROR     ErrorType = "RUNTIME_ERROR"
)

type ErrorDetails struct {
	Id       string    `json:"id,omitempty"`
	Name     string    `json:"name,omitempty"`
	Severity *Severity `json:"severity,omitempty"`
}

type Severity struct {
	Description string  `json:"description,omitempty"`
	Score       float32 `json:"score,omitempty"`
}
