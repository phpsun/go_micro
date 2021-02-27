package bucket

import (
	"common/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
	"path/filepath"
)

type S3Client struct {
	session *session.Session
	bucket  string
}

func NewS3Client(c *BucketConfig) (*S3Client, error) {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, ""),
		Region:           aws.String(c.Region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(false), //virtual-host style
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return nil, err
	}

	return &S3Client{session: sess, bucket: c.Bucket}, nil
}

func (this *S3Client) Upload(filePath string, keyName string, mimeType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := s3manager.NewUploader(this.session)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(this.bucket),
		Key:         aws.String(keyName),
		ContentType: aws.String(mimeType),
		Body:        file,
	})
	return err
}

func (this *S3Client) Download(keyName string, filePath string) error {
	err := util.EnsureDir(filepath.Dir(filePath))
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(this.session)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(this.bucket),
			Key:    aws.String(keyName),
		})
	return err
}

func (this *S3Client) Delete(keyName string) error {
	// Create S3 service client
	svc := s3.New(this.session)

	// Delete the item
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(this.bucket),
		Key:    aws.String(keyName),
	})
	if err != nil {
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(this.bucket),
		Key:    aws.String(keyName),
	})
	if err != nil {
		return err
	}

	return nil
}

func (this *S3Client) Exists(keyName string) error {
	// Create S3 service client
	svc := s3.New(this.session)

	_, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(this.bucket),
		Key:    aws.String(keyName),
	})
	return err
}
