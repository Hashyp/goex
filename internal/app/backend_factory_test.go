package app

import "testing"

func TestPaneBackendChoiceFromPaneGCSBackend(t *testing.T) {
	pane := Pane{
		backend:  NewGCSBackend(nil, "goex", 0),
		location: GCSLocation{Mode: GCSModeBuckets},
	}

	if got := paneBackendChoiceFromPane(pane); got != paneBackendGCS {
		t.Fatalf("expected GCS backend choice, got %v", got)
	}
}

func TestPaneBackendChoiceFromPaneStaticErrorGCSLocation(t *testing.T) {
	pane := Pane{
		backend:  NewStaticErrorBackendWithLocation(nil, GCSLocation{Mode: GCSModeBuckets}, "gcs:///"),
		location: GCSLocation{Mode: GCSModeBuckets},
	}

	if got := paneBackendChoiceFromPane(pane); got != paneBackendGCS {
		t.Fatalf("expected GCS backend choice for static error pane, got %v", got)
	}
}
