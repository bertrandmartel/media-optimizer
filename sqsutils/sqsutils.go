package sqsutils

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func GetQueueURL(sqsSvc *sqs.SQS, queueName *string) *string {
	resultURL, err := sqsSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queueName,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
			fmt.Printf("unable to find queue %v\n", queueName)
		} else {
			fmt.Printf("unable to queue %v, %v\n", queueName, err)
		}
		return nil
	}
	return resultURL.QueueUrl
}

func ReceiveMessages(sqsSvc *sqs.SQS, queueURL *string, timeout *int64) *[]*sqs.Message {
	result, err := sqsSvc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
		AttributeNames: aws.StringSlice([]string{
			"SentTimestamp",
		}),
		MaxNumberOfMessages: aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
		WaitTimeSeconds: timeout,
	})
	if err != nil {
		fmt.Printf("Unable to receive message from queue : %v\n", *queueURL)
		return nil
	}
	return &result.Messages
}
