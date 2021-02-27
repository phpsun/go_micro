package bucket

import "fmt"

type BucketConfig struct {
	Type      string `toml:"type"`
	Region    string `toml:"region"`
	Bucket    string `toml:"bucket"`
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
}

type BucketClient interface {
	Upload(filePath string, keyName string, mimeType string) error
	Download(keyName string, filePath string) error
	Delete(keyName string) error
	Exists(keyName string) error
}

func NewBucketClient(c *BucketConfig) (BucketClient, error) {
	switch c.Type {
	case "local":
		return NewLocalClient(c)
	case "s3":
		return NewS3Client(c)
	default:
		return nil, fmt.Errorf("NewBucketClient, Unknown type: %s", c.Type)
	}
}
