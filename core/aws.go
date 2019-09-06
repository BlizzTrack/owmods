package core

import (
	"crypto/tls"
	"fmt"
	"github.com/minio/minio-go"
	"github.com/blizztrack/owmods/system"
	"gopkg.in/alecthomas/kingpin.v2"
	"image"
	"io"
	"log"
	"net/http"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type awsClient struct {
	bucketName    string
	bucketBaseURL string
	client        *minio.Client
}

var (
	awsClientInstance *awsClient

	bucketName     = kingpin.Flag("bucket_name", "AWS Bucket Name").Envar("AWS_BUCKET_NAME").Default("updater").String()
	bucketEndpoint = kingpin.Flag("bucket_endpoint", "AWS Bucket Endpoint").Envar("AWS_BUCKET_ENDPOINT").Default("sfo2.digitaloceanspaces.com").String()
	bucketKey      = kingpin.Flag("bucket_key", "AWS Bucket key").Envar("AWS_BUCKET_KEY").Default("").String()
	bucketSecret   = kingpin.Flag("bucket_secret", "AWS Bucket Secret").Envar("AWS_BUCKET_SECRET").Default("").String()
	bucketBaseURL  = kingpin.Flag("bucket_base_url", "AWS Bucket Base URL").Envar("AWS_BUCKET_BASE_URL").Default("https://updater.sfo2.cdn.digitaloceanspaces.com/").String()
)

func newAwsClient() *awsClient {
	// Some reason DOs keys don't work on windows half the time
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client, err := minio.New(*bucketEndpoint, *bucketKey, *bucketSecret, true)
	if err != nil {
		log.Println(err)
	}
	client.SetCustomTransport(tr)

	return &awsClient{
		bucketName:    *bucketName,
		client:        client,
		bucketBaseURL: *bucketBaseURL,
	}
}

func AWSClient() *awsClient {
	if awsClientInstance == nil {
		awsClientInstance = newAwsClient()
	}

	return awsClientInstance
}

func (aws *awsClient) PutFile(fileName, fileContentType string, size int64, data io.Reader) (int64, error) {
	userMetaData := map[string]string{"x-amz-acl": "public-read"}

	return aws.client.PutObject(aws.bucketName, fileName, data, size, minio.PutObjectOptions{
		ContentType:  fileContentType,
		UserMetadata: userMetaData,
	})
}

func (aws *awsClient) DeleteFile(fileName string) error {
	return aws.client.RemoveObject(aws.bucketName, fileName)
}

func (aws *awsClient) BaseURL() string {
	return aws.bucketBaseURL
}

func (*awsClient) CreateFileName(ext string) string {
	return fmt.Sprintf("images/%s/%s/%s%s", system.RandomString(5), system.RandomString(5), system.RandomString(32), ext)
}

func (*awsClient) GetImageSize(data io.Reader) (int, int) {
	image, _, err := image.DecodeConfig(data)
	if err != nil {
		log.Println(err)
	}
	return image.Width, image.Height
}
