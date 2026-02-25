package app

import (
	"fmt"
	"time"
)

type TransferOpType string

const (
	TransferOpCopy TransferOpType = "copy"
	TransferOpMove TransferOpType = "move"
)

type TransferConflictPolicy string

const (
	TransferConflictSkip      TransferConflictPolicy = "skip"
	TransferConflictOverwrite TransferConflictPolicy = "overwrite"
)

func (p TransferConflictPolicy) Validate() error {
	switch p {
	case TransferConflictSkip, TransferConflictOverwrite:
		return nil
	default:
		return fmt.Errorf("invalid transfer conflict policy: %q", p)
	}
}

type TransferObjectRef struct {
	Provider string
	Scope    string
	Path     string
	Display  string
}

type TransferObjectMetadata struct {
	SizeBytes  int64
	ModTime    time.Time
	HasModTime bool
}

type TransferPlanItem struct {
	Source      TransferObjectRef
	Destination TransferObjectRef
}

type TransferResultItem struct {
	PlanItem    TransferPlanItem
	BytesCopied int64
}

type TransferFailure struct {
	PlanItem TransferPlanItem
	Stage    string
	Err      error
}

type TransferResult struct {
	Op          TransferOpType
	Copied      []TransferResultItem
	Skipped     []TransferPlanItem
	Failed      []TransferFailure
	BytesCopied int64
}

type TransferProgress struct {
	Op          TransferOpType
	Item        TransferPlanItem
	Index       int
	Total       int
	ItemBytes   int64
	BytesCopied int64
}

type TransferProgressFunc func(TransferProgress)
