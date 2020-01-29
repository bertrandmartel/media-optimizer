package model

type Config struct {
	S3Config   S3Config    `json:"s3Config"`
	IgnoreTag  string      `jons:"ignoreTag"`
	Optimizers []Optimizer `json:"optimizers"`
}

type S3Config struct {
	ACL                  string `json:"acl"`
	StorageClass         string `json:"storageClass"`
	ServerSideEncryption string `json:"serverSideEncryption"`
	CacheControl         string `json:"cacheControl"`
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
