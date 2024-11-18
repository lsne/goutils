/*
 * @Author: lsne
 * @Date: 2023-12-07 19:09:29
 */

package s3ceph

import "time"

type S3Bucket struct {
	Name         string
	CreationDate time.Time
}

type S3Object struct {
	Key          string
	LastModified time.Time
	Size         int64
	StorageClass string
}
