package app

import "testing"

func TestResolveDestinationRefLocal(t *testing.T) {
	ref, err := resolveDestinationRef(LocalLocation{Path: "/tmp/dst"}, "docs/a.txt")
	if err != nil {
		t.Fatalf("resolve destination: %v", err)
	}
	if ref.Provider != "local" {
		t.Fatalf("unexpected provider: %q", ref.Provider)
	}
	if ref.Scope != "/tmp/dst" {
		t.Fatalf("unexpected scope: %q", ref.Scope)
	}
	if ref.Path != "/tmp/dst/docs/a.txt" {
		t.Fatalf("unexpected path: %q", ref.Path)
	}
}

func TestResolveDestinationRefCloudPrefixes(t *testing.T) {
	cases := []struct {
		name      string
		location  Location
		wantProv  string
		wantScope string
		wantPath  string
	}{
		{
			name:      "s3",
			location:  S3Location{Mode: S3ModeObjects, Bucket: "bkt", Prefix: "folder/"},
			wantProv:  "s3",
			wantScope: "bkt",
			wantPath:  "folder/a/b.txt",
		},
		{
			name:      "azure",
			location:  AzureLocation{Mode: AzureModeObjects, Container: "cnt", Prefix: "folder/"},
			wantProv:  "azure",
			wantScope: "cnt",
			wantPath:  "folder/a/b.txt",
		},
		{
			name:      "gcs",
			location:  GCSLocation{Mode: GCSModeObjects, Bucket: "bkt", Prefix: "folder/"},
			wantProv:  "gcs",
			wantScope: "bkt",
			wantPath:  "folder/a/b.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := resolveDestinationRef(tc.location, "a/b.txt")
			if err != nil {
				t.Fatalf("resolve destination: %v", err)
			}
			if ref.Provider != tc.wantProv {
				t.Fatalf("provider mismatch: got %q want %q", ref.Provider, tc.wantProv)
			}
			if ref.Scope != tc.wantScope {
				t.Fatalf("scope mismatch: got %q want %q", ref.Scope, tc.wantScope)
			}
			if ref.Path != tc.wantPath {
				t.Fatalf("path mismatch: got %q want %q", ref.Path, tc.wantPath)
			}
		})
	}
}
