package models

const (
	DestinationTypeLocal = "local"
	DestinationTypeS3    = "s3"
)

type BaseDestination struct {
	Type string `json:"type"`
}

type LocalDestination struct {
	BaseDestination

	Path          string `json:"path"`
	AccessBaseURL string `json:"access_baseurl"`
}

type S3Destination struct {
	BaseDestination

	Path          string `json:"path"`
	Bucket        string `json:"bucket"`
	Endpoint      string `json:"endpoint"`
	SecretID      string `json:"secret_id"`
	SecretKey     string `json:"secret_key"`
	AccessBaseURL string `json:"access_baseurl"`
	EnableSSL     bool   `json:"enable_ssl"`
}
