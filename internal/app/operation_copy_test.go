package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type fakeCopySource struct {
	payloadByPath map[string][]byte
	openErrByPath map[string]error
}

func (f fakeCopySource) OpenCopyReader(_ context.Context, source TransferObjectRef) (CopyReadHandle, error) {
	if err, ok := f.openErrByPath[source.Path]; ok {
		return CopyReadHandle{}, err
	}

	payload := f.payloadByPath[source.Path]
	return CopyReadHandle{
		Reader: io.NopCloser(bytes.NewReader(payload)),
		Metadata: TransferObjectMetadata{
			SizeBytes: int64(len(payload)),
		},
	}, nil
}

type fakeCopyTarget struct {
	existsByPath   map[string]bool
	existsErr      error
	openErrByPath  map[string]error
	closeErrByPath map[string]error
	writes         map[string][]byte
}

func (f *fakeCopyTarget) CopyDestinationExists(_ context.Context, destination TransferObjectRef) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}

	return f.existsByPath[destination.Path], nil
}

func (f *fakeCopyTarget) OpenCopyWriter(_ context.Context, destination TransferObjectRef, _ TransferObjectMetadata) (io.WriteCloser, error) {
	if err, ok := f.openErrByPath[destination.Path]; ok {
		return nil, err
	}
	if f.writes == nil {
		f.writes = map[string][]byte{}
	}

	return &recordingWriteCloser{
		onClose: func(buf []byte) error {
			f.writes[destination.Path] = append([]byte(nil), buf...)
			if err, ok := f.closeErrByPath[destination.Path]; ok {
				return err
			}
			return nil
		},
	}, nil
}

type recordingWriteCloser struct {
	buf     bytes.Buffer
	onClose func([]byte) error
}

func (r *recordingWriteCloser) Write(p []byte) (n int, err error) {
	return r.buf.Write(p)
}

func (r *recordingWriteCloser) Close() error {
	if r.onClose == nil {
		return nil
	}

	return r.onClose(r.buf.Bytes())
}

func TestExecuteCopySkipsOnConflictPolicySkip(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := fakeCopySource{
		payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
	}
	target := &fakeCopyTarget{
		existsByPath: map[string]bool{"dest/alpha.txt": true},
	}

	result := ExecuteCopy(context.Background(), TransferCopyRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictSkip,
	}, source, target)

	if len(result.Copied) != 0 {
		t.Fatalf("expected 0 copied items, got %d", len(result.Copied))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected 0 failures, got %d", len(result.Failed))
	}
}

func TestExecuteCopyOverwritesWhenPolicyIsOverwrite(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := fakeCopySource{
		payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
	}
	target := &fakeCopyTarget{
		existsByPath: map[string]bool{"dest/alpha.txt": true},
	}

	result := ExecuteCopy(context.Background(), TransferCopyRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictOverwrite,
	}, source, target)

	if len(result.Copied) != 1 {
		t.Fatalf("expected 1 copied item, got %d", len(result.Copied))
	}
	if got := string(target.writes["dest/alpha.txt"]); got != "alpha" {
		t.Fatalf("unexpected written payload: %q", got)
	}
	if result.BytesCopied != int64(len("alpha")) {
		t.Fatalf("expected bytes copied %d, got %d", len("alpha"), result.BytesCopied)
	}
}

func TestExecuteCopyRecordsWriteCloseFailure(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := fakeCopySource{
		payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
	}
	target := &fakeCopyTarget{
		existsByPath:   map[string]bool{},
		closeErrByPath: map[string]error{"dest/alpha.txt": errors.New("close failed")},
	}

	result := ExecuteCopy(context.Background(), TransferCopyRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictOverwrite,
	}, source, target)

	if len(result.Copied) != 0 {
		t.Fatalf("expected 0 copied items, got %d", len(result.Copied))
	}
	if len(result.Failed) != 1 {
		t.Fatalf("expected 1 failed item, got %d", len(result.Failed))
	}
	if result.Failed[0].Stage != "copy" {
		t.Fatalf("expected copy failure stage, got %q", result.Failed[0].Stage)
	}
}
