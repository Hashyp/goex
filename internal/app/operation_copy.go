package app

import (
	"context"
	"io"
	"sync"
)

const copyBufferSize = 128 * 1024

var copyBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, copyBufferSize)
	},
}

type CopyReadHandle struct {
	Reader   io.ReadCloser
	Metadata TransferObjectMetadata
}

type TransferCopySource interface {
	OpenCopyReader(ctx context.Context, source TransferObjectRef) (CopyReadHandle, error)
}

type TransferCopyTarget interface {
	CopyDestinationExists(ctx context.Context, destination TransferObjectRef) (bool, error)
	OpenCopyWriter(ctx context.Context, destination TransferObjectRef, metadata TransferObjectMetadata) (io.WriteCloser, error)
}

type TransferCopyRequest struct {
	Plan           []TransferPlanItem
	ConflictPolicy TransferConflictPolicy
	ReportProgress TransferProgressFunc
}

func ExecuteCopy(ctx context.Context, request TransferCopyRequest, source TransferCopySource, target TransferCopyTarget) TransferResult {
	result := TransferResult{
		Op:      TransferOpCopy,
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

		result.Copied = append(result.Copied, TransferResultItem{
			PlanItem:    item,
			BytesCopied: bytesWritten,
		})
		result.BytesCopied += bytesWritten

		if request.ReportProgress != nil {
			request.ReportProgress(TransferProgress{
				Op:          TransferOpCopy,
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

func copySingleItem(ctx context.Context, readHandle CopyReadHandle, item TransferPlanItem, target TransferCopyTarget) (int64, error) {
	defer func() {
		_ = readHandle.Reader.Close()
	}()

	writer, err := target.OpenCopyWriter(ctx, item.Destination, readHandle.Metadata)
	if err != nil {
		return 0, err
	}

	buf := copyBufferPool.Get().([]byte)
	defer copyBufferPool.Put(buf)

	written, copyErr := io.CopyBuffer(writer, readHandle.Reader, buf)
	closeErr := writer.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if closeErr != nil {
		return written, closeErr
	}

	return written, nil
}
