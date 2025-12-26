package minio

// PlatformConfig holds MinIO platform configuration
type PlatformConfig struct {
	EndpointKubernetes string `mapstructure:"MINIO_ENDPOINT_KUBERNETES" validate:"required"`
	EndpointLocal      string `mapstructure:"MINIO_ENDPOINT_LOCAL" validate:"required"`
	AccessKey          string `mapstructure:"MINIO_ACCESS_KEY" validate:"required"`
	SecretKey          string `mapstructure:"MINIO_SECRET_KEY" validate:"required"`
	TLSVerify          bool   `mapstructure:"MINIO_TLS_VERIFY"`
}
