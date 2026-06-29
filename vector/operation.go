package vector

import (
	"encoding/json"
	"time"
)

// ReplicatedOperationKind describes the subsystem that owns a replicated
// operation. SQLite state operations require strict leader forwarding, while
// vector operations can keep their existing best-effort behavior.
type ReplicatedOperationKind string

const (
	ReplicatedOperationKindVector ReplicatedOperationKind = "vector"
	ReplicatedOperationKindSQLite ReplicatedOperationKind = "sqlite"
)

// ReplicatedOperation is the common envelope for cluster-replicated state
// transitions. The payload schema is defined by Type and Kind.
type ReplicatedOperation struct {
	ID        string                  `json:"id,omitempty"`
	Kind      ReplicatedOperationKind `json:"kind"`
	Type      string                  `json:"type"`
	Strict    bool                    `json:"strict,omitempty"`
	Payload   json.RawMessage         `json:"payload,omitempty"`
	RaftTerm  uint64                  `json:"raftTerm,omitempty"`
	LogIndex  uint64                  `json:"logIndex,omitempty"`
	CreatedAt time.Time               `json:"createdAt,omitempty"`
}

// ApplyFunc applies a non-vector replicated operation on the receiving node.
type ApplyFunc func(ReplicatedOperation) error

// WrapVectorOperation converts the legacy vector operation into the common
// replicated operation envelope.
func WrapVectorOperation(op Operation) (ReplicatedOperation, error) {
	payload, err := json.Marshal(op)
	if err != nil {
		return ReplicatedOperation{}, err
	}
	return ReplicatedOperation{
		Kind:      ReplicatedOperationKindVector,
		Type:      string(op.Type),
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// UnwrapVectorOperation decodes the legacy vector operation payload.
func UnwrapVectorOperation(op ReplicatedOperation) (Operation, error) {
	var result Operation
	if len(op.Payload) == 0 {
		return result, nil
	}
	return result, json.Unmarshal(op.Payload, &result)
}
