package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	// "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"os"
)

// DeviceUpdateEvent defines the request structure of this user creation request
type DeviceUpdateEvent struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// Response defines the response structure to this user creation request
type Response struct {
	Message string `json:"Response"`
}

// UpdateDevice is the lambda function handler
// it processes the creation of the cognito user
func UpdateDevice(ctx context.Context, evt DeviceRegEvent) (Response, error) {

	sess := session.Must(session.NewSession())

	dynamoService := dynamodb.New(sess)

	// TODO: dynamodbattribute.MarshalMap does not work!? Need to figure out why, for now, we'll make the
	// required structs manually

	macAttributeValue := dynamodb.AttributeValue{
		S: evt.MAC,
	}

	nameAttributeValue := dynamodb.AttributeValue{
		S: evt.Name,
	}

	var dynamoInputItem map[string]*dynamodb.AttributeValue

	dynamoInputItem = make(map[string]*dynamodb.AttributeValue)

	dynamoInputItem["MAC"] = &uuidAttributeValue
	dynamoInputItem["Name"] = &uuidAttributeValue

	dynamoInput := dynamodb.PutItemInput{
		TableName: aws.String("users"),
		Item:      dynamoInputItem,
	}

	_, err = dynamoService.PutItem(&dynamoInput)
	if err != nil {
		return Response{Message: "Error creating dynamo user entry: "}, err
	}

	return Response{Message: fmt.Sprintf("Successfully created user %s", evt.Devicename)}, nil
}

func main() {
	if os.Getenv("AWS_PROFILE") != "" {
		log.Printf("Using AWS Profile: %s\n", os.Getenv("AWS_PROFILE"))
	} else {
		log.Println("Using AWS Profile: default")
	}

	if os.Getenv("AWS_REGION") == "" {
		log.Fatal("AWS_REGION not set")
	}

	lambda.Start(UpdateDevice)
}
