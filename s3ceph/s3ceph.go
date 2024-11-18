/*
 * @Author: lsne
 * @Date: 2023-12-07 19:09:46
 */

package s3ceph

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
)

type S3Ceph struct {
	EndPoint  string
	AccessKey string
	SecretKey string
	Config    *aws.Config
	Session   *session.Session
}

func NewS3Ceph(endPoint, accessKey, secretKey string) (*S3Ceph, error) {
	var err error
	c := &S3Ceph{
		EndPoint:  endPoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	// if !strings.HasPrefix(bucket, "/") {
	// 	Bucket = "/" + bucket
	// }

	c.Config = &aws.Config{
		Credentials:      credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, ""),
		Endpoint:         aws.String(c.EndPoint),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true), //virtual-host style方式，不要修改
	}

	if c.Session, err = session.NewSession(c.Config); err != nil {
		return nil, err
	}

	// c.Session = session.Must(session.NewSessionWithOptions(session.Options{
	// 	Config:            *c.Config,
	// 	SharedConfigState: session.SharedConfigEnable,
	// }))

	return c, nil
}

// 创建 bucket
func (c *S3Ceph) CreateBuckets(bucketName string) error {

	s3Input := &s3.CreateBucketInput{Bucket: aws.String(bucketName)}

	svc := s3.New(c.Session)
	if _, err := svc.CreateBucket(s3Input); err != nil {
		return err
	}

	// Wait until bucket is created before finishing
	// fmt.Printf("Waiting for bucket %q to be created...\n", bucket)

	if err := svc.WaitUntilBucketExists(&s3.HeadBucketInput{Bucket: aws.String(bucketName)}); err != nil {
		return err
	}

	fmt.Printf("Bucket %q successfully created\n", bucketName)
	return nil

}

// 查询 bucket 列表
func (c *S3Ceph) ListBuckets() (result []S3Bucket, err error) {

	svc := s3.New(c.Session)
	rst, err := svc.ListBuckets(nil)
	if err != nil {
		return result, fmt.Errorf("unable to list buckets, %v", err)
	}

	for _, b := range rst.Buckets {
		result = append(result, S3Bucket{
			Name:         aws.StringValue(b.Name),
			CreationDate: aws.TimeValue(b.CreationDate),
		})
	}

	return result, nil

}

// 查询 object 列表
func (c *S3Ceph) ListObjectFromBucket(bucketName string, objectPrefix string) (result []S3Object, err error) {

	s3Input := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(objectPrefix),
	}

	svc := s3.New(c.Session)
	resp, err := svc.ListObjects(s3Input)

	if err != nil {
		return result, err
	}

	for _, item := range resp.Contents {
		result = append(result, S3Object{
			Key:          aws.StringValue(item.Key),
			LastModified: aws.TimeValue(item.LastModified),
			Size:         aws.Int64Value(item.Size),
			StorageClass: aws.StringValue(item.StorageClass),
		})
	}

	return result, nil
}

// Upload 上传文件
func (c *S3Ceph) UploadFile(bucketName string, fileName string, objectName string) error {

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	s3Input := &s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
		Body:   file,
	}

	uploader := s3manager.NewUploader(c.Session)

	if _, err = uploader.Upload(s3Input); err != nil {
		return err
	}
	return nil
}

// 下载文件
func (c *S3Ceph) DownloadFile(bucketName string, objectName string, fileName string) error {

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	s3Input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}

	downloader := s3manager.NewDownloader(c.Session)

	if _, err := downloader.Download(file, s3Input); err != nil {
		return err
	}

	return err
}

// 删除文件
func (c *S3Ceph) DeleteObject(bucketName string, objectName string) error {

	s3DeleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}

	s3HeadObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	}

	svc := s3.New(c.Session)

	if _, err := svc.DeleteObject(s3DeleteInput); err != nil {
		return err
	}

	if err := svc.WaitUntilObjectNotExists(s3HeadObjectInput); err != nil {
		return err
	}
	return nil
}

// 生成预签名URL
func (c *S3Ceph) GenerateGetPresign(bucketName string, objectName string, resConTentType string, expireTime time.Duration) (string, error) {

	// 创建一个 S3 客户端对象
	svc := s3.New(c.Session)

	// 设置预签名URL的参数
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:              aws.String(bucketName),
		Key:                 aws.String(objectName),
		ResponseContentType: aws.String(resConTentType),
	})

	//  构建请求, 这两个函数以后用到了再研究他们的作用和使用方法
	// req.Build()
	// req.PresignRequest()

	// 生成预签名URL, 必须传入一个过期时间
	return req.Presign(expireTime)
}

// 生成预签名URL
func (c *S3Ceph) GeneratePutPresign(bucketName string, objectName string, expireTime time.Duration) (string, error) {

	// 创建一个 S3 客户端对象
	svc := s3.New(c.Session)

	// 设置预签名URL的参数
	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectName),
	})

	// 生成预签名URL, 必须传入一个过期时间
	return req.Presign(expireTime)
}

// GenerateS3StsToken 生成 STS 临时 token
// 前提1: 需要 ceph rgw 开启 sts 支持的两个参数: rgw_sts_key  和 rgw_s3_auth_use_sts 并重启 rgw
// 前提2: 需要管理员对当前用户创建角色: 创建角色: radosgw-admin role create --role-name=myuser01-assume
func (c *S3Ceph) GenerateS3StsToken(tokenName string, roleName string, policy string, DurationSeconds int64) (*sts.AssumeRoleOutput, error) {

	// roleName 示例
	// roleName = "arn:aws:iam:::role/SkylarSaml"

	// policy 示例
	// policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"s3:PutObject\", \"s3:ListAllMyBuckets\"],\"Resource\":[\"arn:aws:s3:::testbucket*/images/*.jpg\", \"arn:aws:s3:::testbucket*/images/*.jpeg\", \"arn:aws:s3:::testbucket*/files/*.exe\"]}]}"

	// 创建一个 S3 客户端对象
	svc := sts.New(c.Session)
	s3Input := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(DurationSeconds),
		Policy:          aws.String(policy),
		RoleArn:         aws.String(roleName),
		RoleSessionName: aws.String(tokenName),
	}

	return svc.AssumeRole(s3Input)
}
