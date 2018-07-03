package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	// "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"os"
)

// DeviceRegEvent defines the request structure of this device registration request
type DeviceRegEvent struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	// The Owner is the user who owns the device
	// See the users table
	// This value should be an email address
	Owner string `json:"owner"`
}

// Response defines the response structure to this device registration request
type Response struct {
	Message string `json:"Response"`
}

// CreateDevice is the lambda function handler
func CreateDevice(ctx context.Context, evt DeviceRegEvent) (Response, error) {

	sess := session.Must(session.NewSession())

	dynamoService := dynamodb.New(sess)

	// TODO: dynamodbattribute.MarshalMap does not work!? Need to figure out why, for now, we'll make the
	// required structs manually

	macAttributeValue := dynamodb.AttributeValue{
		S: &evt.MAC,
	}

	nameAttributeValue := dynamodb.AttributeValue{
		S: &evt.Name,
	}

	ownerAttributeValue := dynamodb.AttributeValue{
		S: &evt.Owner,
	}

	statusAttributeValue := dynamodb.AttributeValue{
		S: aws.String("offline"),
	}

	var dynamoInputItem map[string]*dynamodb.AttributeValue

	dynamoInputItem = make(map[string]*dynamodb.AttributeValue)

	dynamoInputItem["MAC"] = &macAttributeValue
	dynamoInputItem["Name"] = &nameAttributeValue
	dynamoInputItem["Owner"] = &ownerAttributeValue
	dynamoInputItem["Status"] = &statusAttributeValue

	dynamoInput := dynamodb.PutItemInput{
		ConditionExpression: aws.String("attribute_not_exists(MAC)"),
		TableName:           aws.String("devices"),
		Item:                dynamoInputItem,
	}

	_, err := dynamoService.PutItem(&dynamoInput)
	if err != nil {
		return Response{Message: "Error creating dynamo device entry: "}, err
	}

	/*
		var userDeviceMap map[string]*dynamodb.AttributeValue
		userDeviceMap = make(map[string]*dynamodb.AttributeValue)

		trueBool := true

		trueAttributeValue := dynamodb.AttributeValue{
			BOOL: &trueBool,
		}

		userDeviceMap[evt.MAC] = &trueAttributeValue

		userDeviceAttributeValue := dynamodb.AttributeValue{
			M: userDeviceMap,
		}
	*/

	lc, _ := lambdacontext.FromContext(ctx)

	log.Printf("%+v\n", lc)

	return Response{Message: fmt.Sprintf("Successfully registered device %s", evt.MAC)}, nil
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

	lambda.Start(CreateDevice)
}
