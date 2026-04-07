package export

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// newMinioClient creates a new MinIO client from config.
func newMinioClient(cfg ExportConfig) (*minio.Client, error) {
	return minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
}

// objectKey returns the MinIO object key for an export file.
// Format: module/YYYY/MM/exportCode.xlsx
func objectKey(module string, exportCode string) string {
	now := time.Now()
	return fmt.Sprintf("%s/%d/%02d/%s.xlsx", module, now.Year(), now.Month(), exportCode)
}

// uploadToMinIO uploads data to MinIO and returns the object key.
func uploadToMinIO(ctx context.Context, client *minio.Client, bucket, key string, data []byte) error {
	reader := bytes.NewReader(data)
	_, err := client.PutObject(ctx, bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	})
	return err
}

// downloadFromMinIO downloads a file from MinIO.
func downloadFromMinIO(ctx context.Context, client *minio.Client, bucket, key string) ([]byte, error) {
	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	return io.ReadAll(obj)
}

// ensureBucket creates the bucket if it doesn't exist.
func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}
	return nil
}
