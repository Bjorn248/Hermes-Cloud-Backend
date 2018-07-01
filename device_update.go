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

// DeviceUpdateEvent defines the request structure of this device update request
type DeviceUpdateEvent struct {
	MAC    string `json:"mac"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Response defines the response structure to this device update request
type Response struct {
	Message string `json:"Response"`
}

// UpdateDevice is the lambda function handler
func UpdateDevice(ctx context.Context, evt DeviceUpdateEvent) (Response, error) {

	sess := session.Must(session.NewSession())

	dynamoService := dynamodb.New(sess)

	// TODO: dynamodbattribute.MarshalMap does not work!? Need to figure out why, for now, we'll make the
	// required structs manually

	macAttributeValue := dynamodb.AttributeValue{
		S: &evt.MAC,
	}

	var dynamoUpdateKey map[string]*dynamodb.AttributeValue

	dynamoUpdateKey = make(map[string]*dynamodb.AttributeValue)

	dynamoUpdateKey["MAC"] = &macAttributeValue

	// TODO add cognito token and MAC verification at the beginning of
	// this function

	var dynamoUpdateExpressionString string

	dynamoUpdateExpressionString = "SET"

	var expressionAttributeNames map[string]*string
	var expressionAttributeValues map[string]*dynamodb.AttributeValue

	Name := "Name"
	Status := "Status"

	statusAttributeValue := dynamodb.AttributeValue{
		S: &evt.Status,
	}

	nameAttributeValue := dynamodb.AttributeValue{
		S: &evt.Name,
	}

	if evt.Name != "" && evt.Status != "" {
		dynamoUpdateExpressionString = "SET #N = :n, #S = :s"
		expressionAttributeNames = map[string]*string{
			"#N": &Name,
			"#S": &Status,
		}
		expressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":n": &nameAttributeValue,
			":s": &statusAttributeValue,
		}
	} else if evt.Name != "" {
		dynamoUpdateExpressionString = "SET #N = :n"
		expressionAttributeNames = map[string]*string{
			"#N": &Name,
		}
		expressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":n": &nameAttributeValue,
		}
	} else if evt.Status != "" {
		dynamoUpdateExpressionString = "SET #S = :s"
		expressionAttributeNames = map[string]*string{
			"#S": &Status,
		}
		expressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":s": &statusAttributeValue,
		}
	}

	dynamoInput := dynamodb.UpdateItemInput{
		TableName:                 aws.String("devices"),
		Key:                       dynamoUpdateKey,
		UpdateExpression:          aws.String(dynamoUpdateExpressionString),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	}

	_, err := dynamoService.UpdateItem(&dynamoInput)
	if err != nil {
		return Response{Message: "Error updating dynamo device entry: "}, err
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

	return Response{Message: fmt.Sprintf("Successfully updated device %s", evt.MAC)}, nil
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
