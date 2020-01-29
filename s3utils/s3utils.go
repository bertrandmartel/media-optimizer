package s3utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func UploadToS3(
	s3Svc *s3.S3, fileName *string, bucketName *string, object *string,
	ignoreTag *string, contentType *string, acl *string, cacheControl *string) {
	data, err := ioutil.ReadFile(*fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	input := &s3.PutObjectInput{
		Body:                 bytes.NewReader(data),
		Bucket:               aws.String(*bucketName),
		Key:                  aws.String(*object),
		ServerSideEncryption: aws.String("AES256"),
		StorageClass:         aws.String("STANDARD"),
		ContentType:          aws.String(*contentType),
		ACL:                  aws.String(*acl),
		CacheControl:         aws.String(*cacheControl),
		Tagging:              aws.String(fmt.Sprintf("%v=true", *ignoreTag)),
	}
	_, err = s3Svc.PutObject(input)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func GetHeadObject(s3Svc *s3.S3, bucketName *string, object *string) *s3.HeadObjectOutput {
	s3HeadInput := &s3.HeadObjectInput{
		Bucket: aws.String(*bucketName),
		Key:    aws.String(*object),
	}
	s3HeadOutput, err := s3Svc.HeadObject(s3HeadInput)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return s3HeadOutput
}

func IsIgnored(tagSet *[]*s3.Tag, ignoreTag *string) bool {
	tags := *tagSet
	for i := 0; i < len(tags); i++ {
		fmt.Println(*tags[i].Key)
		if *tags[i].Key == *ignoreTag {
			return true
		}
	}
	return false
}

func GetObjectTags(s3Svc *s3.S3, bucketName *string, object *string) *[]*s3.Tag {
	input := &s3.GetObjectTaggingInput{
		Bucket: aws.String(*bucketName),
		Key:    aws.String(*object),
	}

	result, err := s3Svc.GetObjectTagging(input)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return &result.TagSet
}

func DownloadObject(
	s3Svc *s3.S3,
	bucketName *string,
	object *string,
	filePath *string) error {
	input := &s3.GetObjectInput{
		Bucket: aws.String(*bucketName),
		Key:    aws.String(*object),
	}

	result, err := s3Svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return err
	}
	outFile, err := os.Create(*filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, result.Body)
	if err != nil {
		return err
	}
	return nil
}
