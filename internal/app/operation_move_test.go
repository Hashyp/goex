package app

import (
	"context"
	"errors"
	"testing"
)

type fakeMoveSource struct {
	fakeCopySource
	deleteErrByPath map[string]error
	deleted         []string
}

func (f *fakeMoveSource) DeleteTransferSource(_ context.Context, source TransferObjectRef) error {
	if err, ok := f.deleteErrByPath[source.Path]; ok {
		return err
	}
	f.deleted = append(f.deleted, source.Path)
	return nil
}

func TestExecuteMoveDeletesSourceAfterSuccessfulCopy(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := &fakeMoveSource{
		fakeCopySource: fakeCopySource{
			payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
		},
	}
	target := &fakeCopyTarget{
		existsByPath: map[string]bool{},
	}

	result := ExecuteMove(context.Background(), TransferMoveRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictOverwrite,
	}, source, target)

	if len(result.Copied) != 1 {
		t.Fatalf("expected 1 moved item, got %d", len(result.Copied))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected no failures, got %d", len(result.Failed))
	}
	if len(source.deleted) != 1 || source.deleted[0] != "alpha.txt" {
		t.Fatalf("expected alpha.txt to be deleted from source, got %v", source.deleted)
	}
}

func TestExecuteMoveSkipsDeleteWhenConflictPolicySkips(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := &fakeMoveSource{
		fakeCopySource: fakeCopySource{
			payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
		},
	}
	target := &fakeCopyTarget{
		existsByPath: map[string]bool{"dest/alpha.txt": true},
	}

	result := ExecuteMove(context.Background(), TransferMoveRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictSkip,
	}, source, target)

	if len(result.Skipped) != 1 {
		t.Fatalf("expected one skipped item, got %d", len(result.Skipped))
	}
	if len(source.deleted) != 0 {
		t.Fatalf("expected no source deletions on skip, got %v", source.deleted)
	}
}

func TestExecuteMoveReportsDeleteSourceFailure(t *testing.T) {
	plan := []TransferPlanItem{
		{
			Source:      TransferObjectRef{Path: "alpha.txt"},
			Destination: TransferObjectRef{Path: "dest/alpha.txt"},
		},
	}
	source := &fakeMoveSource{
		fakeCopySource: fakeCopySource{
			payloadByPath: map[string][]byte{"alpha.txt": []byte("alpha")},
		},
		deleteErrByPath: map[string]error{"alpha.txt": errors.New("delete failed")},
	}
	target := &fakeCopyTarget{
		existsByPath: map[string]bool{},
	}

	result := ExecuteMove(context.Background(), TransferMoveRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictOverwrite,
	}, source, target)

	if len(result.Copied) != 0 {
		t.Fatalf("expected no moved items on delete failure, got %d", len(result.Copied))
	}
	if len(result.Failed) != 1 {
		t.Fatalf("expected one failure, got %d", len(result.Failed))
	}
	if stage := result.Failed[0].Stage; stage != "delete-source" {
		t.Fatalf("expected delete-source failure stage, got %q", stage)
	}
}
