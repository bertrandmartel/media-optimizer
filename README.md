# Media Optimizer for AWS S3

[![Build Status](https://github.com/bertrandmartel/media-optimizer/workflows/build%20and%20deploy/badge.svg)](https://github.com/bertrandmartel/media-optimizer/actions?workflow=build%20and%20deploy)
[![Go Report Card](https://goreportcard.com/badge/github.com/bertrandmartel/media-optimizer)](https://goreportcard.com/report/github.com/bertrandmartel/media-optimizer)
[![License](http://img.shields.io/:license-mit-blue.svg)](LICENSE.md)

Automatically optimize your images & videos hosted on AWS S3 when an S3 event is detected

This project is inspired by [image-optimizer](https://github.com/spatie/image-optimizer) & [Lambda ECS Worker Pattern](https://github.com/aws-samples/lambda-ecs-worker-pattern)

## Architecture

![architecture](https://user-images.githubusercontent.com/5183022/73283141-f36b2d80-41f2-11ea-88b9-9cd00e9e9750.png)

A tag is also applied to the optimized file so this file will be ignored in subsequent event

## How it works ?

* S3
  * S3 is configured to trigger a specific lambda on objectCreate event
* Lambda
  * catch the S3 Event
  * push the event to a SQS queue
* EC2
  * constantly listens to messages from SQS queue
  * when a message arrives :
    * get the S3 Event from the message
    * get information about S3 object (content-type)
    * download the S3 object
    * executes optimization chain on object (one or more command)
    * upload the optimized object to S3

## Run on AWS

In AWS dashboard, go to cloudformation & create a new stack from the template file [cloudformation.yml](https://github.com/bertrandmartel/media-optimizer/blob/master/cloudformation.yml)

In the parameters, change the bucket name & the subnet value (a private subnet is fine)

![parameters](https://user-images.githubusercontent.com/5183022/73375742-8cb24680-42bc-11ea-8dcb-adb1ac80f0e0.png)

When the stack is up, go to the created S3 bucket & upload some images. After some moment, you will notice they will be automatically optimized

You can check that the new optimized image has the tag `optimizer_ignore` :

![tags](https://user-images.githubusercontent.com/5183022/73320552-f8a29980-423f-11ea-99c5-aa3a16a29d45.png)

In CloudWatch you can get the logs of your Lambda & your docker container running the optimization program

## Optimization configuration

The optimization configuration is located in optimizer.json file. In the actual version, the JSON file need to be at the same level as the media-optimizer executable

### Current configuration

Currently using all the optimizers from [image-optimizer](https://github.com/spatie/image-optimizer) project + ffmpeg for mp4 video :

* image/png
  * `pngquant --output [output_file] -f [intput_file]`
  * `optipng -out [output_file] -i0 -o2 -clobber [intput_file]`
* image/jpeg
  * `jpegoptim -d [output_directory] -m85 --strip-all --all-progressive [intput_file]`
* image/svg+xml
  * `svgo -o [output_file] --disable={cleanupIDs,removeViewBox} [intput_file]`
* image/gif
  * `gifsicle -o [output_file] -b -O3 [intput_file]`
* image/webp
  * `cwebp -o [output_file] -m 6 -pass 10 -mt -q 80 [intput_file]`
* video/mp4
  * `ffmpeg -i [input_file] -vcodec libx264 -crf 24 -y [output_file]`

You can add more format & more commands to the config file `optimizer.json`

### Configuration Format

```json
{	
	"optimizers": [{
		"contentType": "image/png",
		"exec": [{
			"binary": "pngquant",
			"outputFile": "--output",
			"params": ["-f"]
		},{
			"binary":"optipng",
			"outputFile": "-out",
			"params": ["-i0","-o2", "-clobber"]
		}]
	.....
}
```

* `binary` is the command to execute
* `params` are the command parameters
* `outputFile` is the parameter for the output file
* `outputDirectory` is the parameter for the output directory if `outputFile` is not specified
* `inputFile` is the parameter for the input file if neither `outputFile` nor `outputDirectory` are specified. In case `inputFile` is specified the outputFile value will be appended to the parameters (at the end)

## Run locally

```bash
make install
make run
```

## Run with docker (locally)

```bash
docker build . -t media-optimizer
docker run -v $HOME/.aws:/root/.aws -it media-optimizer
```

## Run with docker (DockerHub)

```bash
docker run -v $HOME/.aws:/root/.aws -it bertrandmartel/media-optimizer:latest
```

## Dependencies

* https://github.com/aws/aws-sdk-go
* https://github.com/aws/aws-lambda-go