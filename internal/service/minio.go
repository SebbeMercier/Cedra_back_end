package services

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

func ConnectMinio() {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // ⚠️ à passer à true si tu as HTTPS
	})
	if err != nil {
		log.Println("⚠️ MinIO non configuré :", err)
		return
	}
	MinioClient = client
	log.Println("✅ Connecté à MinIO :", endpoint)
}

func UploadFile(bucket string, file *multipart.FileHeader) (string, error) {
	if MinioClient == nil {
		return "", fmt.Errorf("MinIO non initialisé")
	}
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = MinioClient.PutObject(context.Background(), bucket, file.Filename, f, file.Size,
		minio.PutObjectOptions{ContentType: file.Header.Get("Content-Type")})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("http://%s/%s/%s", os.Getenv("MINIO_ENDPOINT"), bucket, file.Filename)
	return url, nil
}
