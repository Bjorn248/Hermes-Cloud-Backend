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
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"regexp"
	// "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"os"
)

// UserRegEvent defines the request structure of this user creation request
type UserRegEvent struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response defines the response structure to this user creation request
type Response struct {
	Message string `json:"Response"`
	Error   string `json:"Error"`
}

// CreateUser is the lambda function handler
// it processes the creation of the cognito user
func CreateUser(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var cognitoAppClientID string
	if os.Getenv("COGNITO_APP_CLIENT_ID") == "" {
		log.Fatal("COGNITO_APP_CLIENT_ID not set")
	} else {
		cognitoAppClientID = os.Getenv("COGNITO_APP_CLIENT_ID")
	}

	var evt UserRegEvent
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

	if evt.Email == "" {
		resp := Response{
			Message: "email missing from request JSON",
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

	if evt.Password == "" {
		resp := Response{
			Message: "password missing from request JSON",
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

	// Validate the Email
	validEmail, err := regexp.MatchString("(^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+.[a-zA-Z0-9-.]+$)", evt.Email)
	if validEmail == false {
		resp := Response{
			Message: fmt.Sprintf("Invalid MAC Address Provided: %s", evt.Email),
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

	cognitoService := cognitoidentityprovider.New(sess)
	dynamoService := dynamodb.New(sess)

	cognitoInput := cognitoidentityprovider.SignUpInput{
		ClientId: &cognitoAppClientID,
		Username: &evt.Email,
		Password: &evt.Password,
	}

	cognitoResponse, err := cognitoService.SignUp(&cognitoInput)
	if err != nil {
		log.Println("Error Creating User (cognito):", err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "UsernameExistsException":
				resp := Response{
					Message: fmt.Sprintf("Email is already registered: %s", evt.Email),
					Error:   "User Creation Error",
				}
				marshalledResponse, err := json.Marshal(resp)
				if err != nil {
					log.Println("Error marshalling response:", resp)
					panic(err)
				}
				return events.APIGatewayProxyResponse{
					Body:       string(marshalledResponse),
					StatusCode: 409,
				}, nil
			default:
				resp := Response{
					Message: "Error creating cognito user",
					Error:   "User Creation Error",
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

	userUUID := cognitoResponse.UserSub

	// TODO: dynamodbattribute.MarshalMap does not work!? Need to figure out why, for now, we'll make the
	// required structs manually

	uuidAttributeValue := dynamodb.AttributeValue{
		S: userUUID,
	}

	var dynamoInputItem map[string]*dynamodb.AttributeValue

	dynamoInputItem = make(map[string]*dynamodb.AttributeValue)

	dynamoInputItem["userID"] = &uuidAttributeValue

	dynamoInput := dynamodb.PutItemInput{
		TableName: aws.String("users"),
		Item:      dynamoInputItem,
	}

	_, err = dynamoService.PutItem(&dynamoInput)
	if err != nil {
		log.Println("Error Creating User (dynamo):", err)
		resp := Response{
			Message: "Error creating user",
			Error:   "User Creation Error",
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

	resp := Response{
		Message: fmt.Sprintf("Successfully created user %s", evt.Email),
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

	lambda.Start(CreateUser)
}
