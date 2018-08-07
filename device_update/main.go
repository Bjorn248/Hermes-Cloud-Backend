package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
	"os"
	"regexp"
	"unicode/utf8"
)

// DeviceUpdateEvent defines the request structure of this device update request
type DeviceUpdateEvent struct {
	MAC    string `json:"mac"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Device describes the schema of the returned dynamo object
type Device struct {
	MAC    string `json:"mac"`
	Name   string `json:"name"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

// Response defines the response structure to this device update request
type Response struct {
	Message string `json:"Response"`
	Error   string `json:"Error"`
}

// UpdateDevice is the lambda function handler
func UpdateDevice(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var evt DeviceUpdateEvent
	err := json.Unmarshal([]byte(req.Body), &evt)
	if err != nil {
		resp := Response{
			Message: "Error unmarshalling request body",
			Error:   err.Error(),
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 400,
		}, nil
	}

	if evt.MAC == "" {
		resp := Response{
			Message: "mac missing from request JSON",
			Error:   "Invalid Request",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 400,
		}, nil
	}

	// Validate the MAC
	validMAC, err := regexp.MatchString("^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$", evt.MAC)
	if validMAC == false {
		resp := Response{
			Message: fmt.Sprintf("Invalid MAC Address Provided: %s", evt.MAC),
			Error:   "Invalid Request",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 400,
		}, nil
	}

	// Validate the Device Name
	// Needs to be 50 characters or less
	if evt.Name != "" {
		if utf8.RuneCountInString(evt.Name) > 50 {
			resp := Response{
				Message: "Provided name too long",
				Error:   "Invalid Request",
			}
			marshalledResponse, err := json.Marshal(resp)
			if err != nil {
				log.Println("Error marshalling response:", resp)
				panic(err)
			}
			return events.APIGatewayProxyResponse{
				Body:       string(marshalledResponse),
				StatusCode: 400,
			}, nil
		}
	}

	// Validate the status
	// Needs to be 'offline' or 'online'
	if evt.Status != "" {
		if evt.Status != "offline" && evt.Status != "online" {
			resp := Response{
				Message: "status can only have value 'offline' or 'online'",
				Error:   "Invalid Request",
			}
			marshalledResponse, err := json.Marshal(resp)
			if err != nil {
				log.Println("Error marshalling response:", resp)
				panic(err)
			}
			return events.APIGatewayProxyResponse{
				Body:       string(marshalledResponse),
				StatusCode: 400,
			}, nil
		}
	}

	authorizer := req.RequestContext.Authorizer
	if authorizer["claims"] == "" {
		resp := Response{
			Message: "No authorization token provided",
			Error:   "Missing token",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 401,
		}, nil
	}
	typedAuthorizer, ok := authorizer["claims"].(map[string]interface{})
	if ok != true {
		resp := Response{
			Message: "Error getting authorization information from cognito token",
			Error:   "Error unmarshaling request context",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 500,
		}, nil
	}

	// This is the email address provided by the JWT
	// in the request
	emailFromToken := typedAuthorizer["email"]

	sess := session.Must(session.NewSession())

	dynamoService := dynamodb.New(sess)

	macAttributeValue := dynamodb.AttributeValue{
		S: &evt.MAC,
	}

	var dynamoKey map[string]*dynamodb.AttributeValue

	dynamoKey = make(map[string]*dynamodb.AttributeValue)

	dynamoKey["MAC"] = &macAttributeValue

	consistentRead := true

	dynamoGetInput := dynamodb.GetItemInput{
		TableName:      aws.String("devices"),
		Key:            dynamoKey,
		ConsistentRead: &consistentRead,
	}

	var dynamoResponse *dynamodb.GetItemOutput

	dynamoResponse, err = dynamoService.GetItem(&dynamoGetInput)
	if err != nil {
		log.Println("Error updating device (dynamo)", err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "ResourceNotFoundException":
				resp := Response{
					Message: fmt.Sprintf("MAC not found: %s", evt.MAC),
					Error:   "MAC lookup error",
				}
				marshalledResponse, err := json.Marshal(resp)
				if err != nil {
					log.Println("Error marshalling response:", resp)
					panic(err)
				}
				return events.APIGatewayProxyResponse{
					Body:       string(marshalledResponse),
					StatusCode: 400,
				}, nil
			default:
				resp := Response{
					Message: "Error looking up MAC",
					Error:   "MAC lookup error",
				}
				marshalledResponse, err := json.Marshal(resp)
				if err != nil {
					log.Println("Error marshalling response:", resp)
					panic(err)
				}
				return events.APIGatewayProxyResponse{
					Body:       string(marshalledResponse),
					StatusCode: 500,
				}, nil
			}
		}
	}

	if len(dynamoResponse.Item) == 0 {
		resp := Response{
			Message: fmt.Sprintf("MAC not found: %s", evt.MAC),
			Error:   "MAC lookup error",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{
			Body:       string(marshalledResponse),
			StatusCode: 400,
		}, nil
	}

	// This is the email address associated with the MAC in DynamoDB
	emailFromDynamo := dynamoResponse.Item["Owner"].S

	// This means the person sending the request
	// Does not have a token matching the device owner
	// As reported by dynamo
	if emailFromToken != *emailFromDynamo {
		resp := Response{
			Message: "Not authorized to perform this action",
			Error:   "Not authorized",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{Body: string(marshalledResponse), StatusCode: 403}, nil
	}

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
		Key:                       dynamoKey,
		UpdateExpression:          aws.String(dynamoUpdateExpressionString),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	}

	_, err = dynamoService.UpdateItem(&dynamoInput)
	if err != nil {
		resp := Response{
			Message: "Error updating device",
			Error:   "Something went wrong",
		}
		log.Println("DynamoDB Error", err)
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{Body: string(marshalledResponse), StatusCode: 500}, nil
	}

	resp := Response{
		Message: fmt.Sprintf("Successfully updated device %s", evt.MAC),
	}
	marshalledResponse, err := json.Marshal(resp)
	if err != nil {
		log.Println("Error marshalling response:", resp)
		panic(err)
	}

	return events.APIGatewayProxyResponse{Body: string(marshalledResponse), StatusCode: 204}, nil
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
