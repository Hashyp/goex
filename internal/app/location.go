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
