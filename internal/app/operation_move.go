package app

import "context"

type TransferMoveSource interface {
	TransferCopySource
	CopySourceDeleter
}

type TransferMoveTarget interface {
	TransferCopyTarget
}

type TransferMoveRequest struct {
	Plan           []TransferPlanItem
	ConflictPolicy TransferConflictPolicy
	ReportProgress TransferProgressFunc
}

func ExecuteMove(ctx context.Context, request TransferMoveRequest, source TransferMoveSource, target TransferMoveTarget) TransferResult {
	result := TransferResult{
		Op:      TransferOpMove,
		Copied:  make([]TransferResultItem, 0, len(request.Plan)),
		Skipped: make([]TransferPlanItem, 0),
		Failed:  make([]TransferFailure, 0),
	}

	if request.ConflictPolicy == "" {
		request.ConflictPolicy = TransferConflictSkip
	}
	if err := request.ConflictPolicy.Validate(); err != nil {
		for _, item := range request.Plan {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "validate", Err: err})
		}
		return result
	}

	for idx, item := range request.Plan {
		if err := ctx.Err(); err != nil {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "cancel", Err: err})
			continue
		}

		exists, err := target.CopyDestinationExists(ctx, item.Destination)
		if err != nil {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "exists", Err: err})
			continue
		}
		if exists && request.ConflictPolicy == TransferConflictSkip {
			result.Skipped = append(result.Skipped, item)
			continue
		}

		readHandle, err := source.OpenCopyReader(ctx, item.Source)
		if err != nil {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "open-source", Err: err})
			continue
		}

		bytesWritten, writeErr := copySingleItem(ctx, readHandle, item, target)
		if writeErr != nil {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "copy", Err: writeErr})
			continue
		}

		deleteErr := source.DeleteTransferSource(ctx, item.Source)
		if deleteErr != nil {
			result.Failed = append(result.Failed, TransferFailure{PlanItem: item, Stage: "delete-source", Err: deleteErr})
			continue
		}

		result.Copied = append(result.Copied, TransferResultItem{
			PlanItem:    item,
			BytesCopied: bytesWritten,
		})
		result.BytesCopied += bytesWritten

		if request.ReportProgress != nil {
			request.ReportProgress(TransferProgress{
				Op:          TransferOpMove,
				Item:        item,
				Index:       idx + 1,
				Total:       len(request.Plan),
				ItemBytes:   bytesWritten,
				BytesCopied: result.BytesCopied,
			})
		}
	}

	return result
}
