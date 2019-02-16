package api

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ApiDb struct {
	*dynamodb.DynamoDB
}

type EnvSingleton struct {
	db	*ApiDb
}
