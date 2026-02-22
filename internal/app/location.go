package app

type Location interface {
	isLocation()
}

type LocalLocation struct {
	Path string
}

func (LocalLocation) isLocation() {}

type AzureMode string

const (
	AzureModeContainers AzureMode = "containers"
	AzureModeObjects    AzureMode = "objects"
)

type AzureLocation struct {
	Mode      AzureMode
	Container string
	Prefix    string
}

func (AzureLocation) isLocation() {}

type S3Mode string

const (
	S3ModeBuckets S3Mode = "buckets"
	S3ModeObjects S3Mode = "objects"
)

type S3Location struct {
	Mode   S3Mode
	Bucket string
	Prefix string
}

func (S3Location) isLocation() {}

type GCSMode string

const (
	GCSModeBuckets GCSMode = "buckets"
	GCSModeObjects GCSMode = "objects"
)

type GCSLocation struct {
	Mode   GCSMode
	Bucket string
	Prefix string
}

func (GCSLocation) isLocation() {}
