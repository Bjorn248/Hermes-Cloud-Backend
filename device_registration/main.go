package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
	"os"
	"regexp"
	"unicode/utf8"
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
	Error   string `json:"Error"`
}

// CreateDevice is the lambda function handler
func CreateDevice(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var evt DeviceRegEvent
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

	emailFromToken := typedAuthorizer["email"]

	if emailFromToken != evt.Owner {
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

	if evt.Name == "" {
		resp := Response{
			Message: "name missing from request JSON",
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

	if evt.Owner == "" {
		resp := Response{
			Message: "owner missing from request JSON",
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

	_, err = dynamoService.PutItem(&dynamoInput)
	if err != nil {
		resp := Response{
			Message: fmt.Sprintf("The following MAC is already registered: %s", evt.MAC),
			Error:   "Duplicate MAC Error",
		}
		marshalledResponse, err := json.Marshal(resp)
		if err != nil {
			log.Println("Error marshalling response:", resp)
			panic(err)
		}
		return events.APIGatewayProxyResponse{Body: string(marshalledResponse), StatusCode: 409}, nil
	}

	resp := Response{
		Message: fmt.Sprintf("Successfully registered device %s", evt.MAC),
	}
	marshalledResponse, err := json.Marshal(resp)
	if err != nil {
		log.Println("Error marshalling response:", resp)
		panic(err)
	}

	return events.APIGatewayProxyResponse{Body: string(marshalledResponse), StatusCode: 200}, nil
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
