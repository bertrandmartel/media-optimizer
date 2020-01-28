package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bertrandmartel/media-optimizer/fileutils"
	"github.com/bertrandmartel/media-optimizer/s3utils"
	"github.com/bertrandmartel/media-optimizer/sqsutils"
)

const inputMediaDir = "/tmp/input_media"
const outputMediaDir = "/tmp/output_media"
const configFile = "optimizer.json"

type Config struct {
	IgnoreTag  string      `jons:"ignoreTag"`
	Optimizers []Optimizer `json:"optimizers"`
}

type Optimizer struct {
	ContentType string    `json:"contentType"`
	Exec        []Command `json:"exec"`
}

type Command struct {
	Binary          string   `json:"binary"`
	OutputFile      string   `json:"outputFile"`
	OutputDirectory string   `json:"outputDirectory"`
	IntputFile      string   `json:"inputFile"`
	Params          []string `json:"params"`
}

func main() {
	fileutils.InitDir(inputMediaDir)
	fileutils.InitDir(outputMediaDir)

	config := Config{
		Optimizers: []Optimizer{},
	}
	getConfig(&config)

	var queueName = "ProcessMediaQueue"
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	sqsSvc := sqs.New(sess)
	s3Svc := s3.New(sess)

	queueURL := sqsutils.GetQueueURL(sqsSvc, &queueName)

	if queueURL == nil {
		fmt.Println("SQS request failed. Maybe SQS queue is not created or a problem with credentials")
		return
	}
	fmt.Printf("Get queueURL : %v\n", *queueURL)
	go forever(&config, sqsSvc, s3Svc, queueURL)
	select {} // block forever
}

func getConfig(config *Config) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
}

func forever(config *Config, sqsSvc *sqs.SQS, s3Svc *s3.S3, queueURL *string) {
	for {
		process(config, sqsSvc, s3Svc, queueURL)
		fmt.Println("cleaning directories...")
		fileutils.ClearDir(inputMediaDir)
		fileutils.ClearDir(outputMediaDir)
		time.Sleep(time.Second)
	}
}

func process(config *Config, sqsSvc *sqs.SQS, s3Svc *s3.S3, queueURL *string) {
	var timeout int64 = 20
	fmt.Println("Waiting for messages...")
	msgRes := sqsutils.ReceiveMessages(sqsSvc, queueURL, &timeout)
	if msgRes == nil {
		return
	}
	messages := *msgRes
	fmt.Printf("Received %d messages.\n", len(messages))

	for i := 0; i < len(messages); i++ {
		data := *messages[i].Body
		event := &events.S3Event{
			Records: []events.S3EventRecord{},
		}
		err := json.Unmarshal([]byte(data), event)
		if err != nil {
			fmt.Printf("S3 event wasn't parsed properly : %v\n", err)
			return
		}
		fmt.Println(event)
		for j := 0; j < len(event.Records); j++ {
			fmt.Println(event.Records[j])
			bucketName := event.Records[j].S3.Bucket.Name
			object, err := url.QueryUnescape(event.Records[j].S3.Object.Key)
			if err != nil {
				fmt.Println("fail to url decode object key")
				return
			}
			generatedName := fileutils.GenerateFileName(&object)
			inputFilePath := fmt.Sprintf("%v/%v%v", inputMediaDir, generatedName, filepath.Ext(object))
			outputFilePath := fmt.Sprintf("%v/%v%v", outputMediaDir, generatedName, filepath.Ext(object))

			fmt.Printf("bucket name : %v\n", bucketName)
			fmt.Printf("object key  : %v\n", object)
			fmt.Printf("input file path   : %v\n", inputFilePath)
			fmt.Printf("output file path   : %v\n", outputFilePath)

			s3Tags := s3utils.GetObjectTags(s3Svc, &bucketName, &object)

			if s3Tags == nil {
				fmt.Println("tag request failed. Maybe the file doesn't exist anymore. skipping...")
				break
			}

			isIgnored := s3utils.IsIgnored(s3Tags, &config.IgnoreTag)

			if isIgnored {
				fmt.Println("ignore tag detected. skipping...")
				break
			}

			headObject := s3utils.GetHeadObject(s3Svc, &bucketName, &object)

			fmt.Printf("processing optimizer for content type : %v\n", *headObject.ContentType)
			optimizer := getOptimizer(config, headObject.ContentType)

			if optimizer == nil {
				fmt.Printf("optimizer not found for contentType %v\n", *headObject.ContentType)
				break
			}
			s3utils.DownloadObject(s3Svc, &bucketName, &object, &inputFilePath)
			processCommands(&inputFilePath, &outputFilePath, optimizer)
			if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
				fmt.Printf("file %v is not existing\n", outputFilePath)
				break
			}
			fmt.Printf("upload %v to S3\n", outputFilePath)
			s3utils.UploadToS3(s3Svc, &outputFilePath, &bucketName, &object, &config.IgnoreTag)
		}
		_, err = sqsSvc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      queueURL,
			ReceiptHandle: messages[i].ReceiptHandle,
		})
		if err != nil {
			fmt.Println("Delete Error", err)
			return
		}
		fmt.Println("Message Deleted")
	}
}

func getOptimizer(config *Config, contentType *string) *Optimizer {
	for i := 0; i < len(config.Optimizers); i++ {
		if config.Optimizers[i].ContentType == *contentType {
			return &config.Optimizers[i]
		}
	}
	return nil
}

func processCommands(inputFilePath *string, outputFilePath *string, optimizer *Optimizer) {
	optim := *optimizer
	currentInputFilePath := *inputFilePath
	currentOutputFilePath := ""
	for i := 0; i < len(optim.Exec); i++ {
		if i > 0 {
			currentInputFilePath = fmt.Sprintf("%v-%v%v",
				fileutils.GetFilenameWithoutExt(*outputFilePath), i-1, filepath.Ext(*outputFilePath))
		}
		currentOutputFilePath = fmt.Sprintf("%v-%v%v",
			fileutils.GetFilenameWithoutExt(*outputFilePath), i, filepath.Ext(*outputFilePath))
		fmt.Printf("processing %v\n", optim.Exec[i])
		if optim.Exec[i].Binary == "" {
			fmt.Println("binary was not specified, skipping...")
			break
		}
		var cmd *exec.Cmd
		cmdParam := []string{}
		switch {
		case optim.Exec[i].OutputFile != "":
			cmdParam = append(cmdParam, optim.Exec[i].OutputFile)
			cmdParam = append(cmdParam, currentOutputFilePath)
			cmdParam = append(cmdParam, optim.Exec[i].Params...)
			cmdParam = append(cmdParam, currentInputFilePath)
			fmt.Printf("executing %v %v\n", cmdParam, currentInputFilePath)
			cmd = exec.Command(optim.Exec[i].Binary, cmdParam...)
		case optim.Exec[i].OutputDirectory != "":
			cmdParam = append(cmdParam, optim.Exec[i].OutputDirectory)
			cmdParam = append(cmdParam, outputMediaDir)
			cmdParam = append(cmdParam, optim.Exec[i].Params...)
			cmdParam = append(cmdParam, currentInputFilePath)
			fmt.Printf("executing %v %v\n", cmdParam, currentInputFilePath)
			cmd = exec.Command(optim.Exec[i].Binary, cmdParam...)
		case optim.Exec[i].IntputFile != "":
			cmdParam = append(cmdParam, optim.Exec[i].IntputFile)
			cmdParam = append(cmdParam, currentInputFilePath)
			cmdParam = append(cmdParam, optim.Exec[i].Params...)
			cmdParam = append(cmdParam, currentOutputFilePath)
			fmt.Printf("executing %v %v\n", cmdParam, currentInputFilePath)
			cmd = exec.Command(optim.Exec[i].Binary, cmdParam...)
		}
		if cmd == nil {
			fmt.Println("command didn't match any criteria with file parameters, skipping...")
			break
		}
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
			break
		}
		if optim.Exec[i].OutputDirectory != "" {
			fmt.Printf("moving %v to %v\n", *outputFilePath, currentOutputFilePath)
			//move output to current dest
			fileutils.MoveFile(*outputFilePath, currentOutputFilePath)
		}
	}
	fmt.Println(currentOutputFilePath)
	*outputFilePath = currentOutputFilePath
}
