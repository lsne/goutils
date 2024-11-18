/*
 * @Author: lsne
 * @Date: 2023-12-30 15:24:22
 */

package s3ceph

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestNewS3Ceph(t *testing.T) {
	endPoint := os.Getenv("GoTestEndPoint")
	accessKey := os.Getenv("GoTestAccessKey")
	secretKey := os.Getenv("GoTestSecretKey")
	bucketName := os.Getenv("GoTestBucketName")

	fmt.Println(endPoint)

	s3c, err := NewS3Ceph(endPoint, accessKey, secretKey)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 创建 bucket
	// if err := s3c.CreateBuckets(bucketName); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// 列出所有 bucket
	buckets, err := s3c.ListBuckets()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, bucket := range buckets {
		fmt.Println("输出bucket信息:", bucket)
	}

	// 上传
	if err := s3c.UploadFile(bucketName, "../go.sum", "mypath/pro/go.sum"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 上传
	if err := s3c.UploadFile(bucketName, "../go.mod", "mypath/pro/go.mod"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 列出指定 bucket 中指定前缀的对象, 前缀为空表示查询所有对象
	objects, err := s3c.ListObjectFromBucket(bucketName, "mypath/pro/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("数据目录和文件列表:")
	for _, object := range objects {
		fmt.Println(object)
	}

	// 下载
	if err := s3c.DownloadFile(bucketName, "mypath/pro/go.mod", "down_go.mod"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 删除
	// if err := s3c.DeleteObject(bucketName, "mypath/pro/go.sum"); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// s3c.GenerateS3StsToken()

	// 通过预签名 url 上传文件
	url, err := s3c.GeneratePutPresign(bucketName, "lsne_test/lsne_testfile011", time.Duration(30*time.Minute))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 将当前目录的 main 文件上传到 s3
	file, err := os.Open("main")
	if err != nil {
		fmt.Printf("Unable to open file:%v", err)
		os.Exit(1)
	}
	defer file.Close()

	// 构建预签名 url http 请求
	request, _ := http.NewRequest("PUT", url, file)
	// request.Header.Add("Content-Type", "YOUR_CONTENT_TYPE")
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Printf("err(%v)", err)
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))
}
