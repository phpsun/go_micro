package bucket

import (
	"common/util"
	"os"
)

type LocalClient struct {
	rootPath string
}

func NewLocalClient(c *BucketConfig) (*LocalClient, error) {
	path := c.Region + "/" + c.Bucket + "/"
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	return &LocalClient{rootPath: path}, nil
}

func (this *LocalClient) Upload(filePath string, keyName string, mimeType string) error {
	dstPath := this.rootPath + keyName
	return util.CopyFile(filePath, dstPath)
}

func (this *LocalClient) Download(keyName string, filePath string) error {
	srcPath := this.rootPath + keyName
	return util.CopyFile(srcPath, filePath)
}

func (this *LocalClient) Delete(keyName string) error {
	srcPath := this.rootPath + keyName
	return os.Remove(srcPath)
}

func (this *LocalClient) Exists(keyName string) error {
	srcPath := this.rootPath + keyName
	return util.IsFileExists(srcPath)
}
