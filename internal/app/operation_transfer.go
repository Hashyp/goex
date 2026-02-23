package app

func mergeTransferResults(base TransferResult, next TransferResult) TransferResult {
	if base.Op == "" {
		base.Op = next.Op
	}
	base.Copied = append(base.Copied, next.Copied...)
	base.Skipped = append(base.Skipped, next.Skipped...)
	base.Failed = append(base.Failed, next.Failed...)
	base.BytesCopied += next.BytesCopied
	return base
}
