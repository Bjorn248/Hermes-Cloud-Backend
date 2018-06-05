package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	// "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"os"
)

// UserRegEvent defines the request structure of this user creation request
type UserRegEvent struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Response defines the response structure to this user creation request
type Response struct {
	Message string `json:"Response"`
}

// CreateUser is the lambda function handler
// it processes the creation of the cognito user
func CreateUser(ctx context.Context, evt UserRegEvent) (Response, error) {
	var cognitoAppClientID string
	if os.Getenv("COGNITO_APP_CLIENT_ID") == "" {
		log.Fatal("COGNITO_APP_CLIENT_ID not set")
	} else {
		cognitoAppClientID = os.Getenv("COGNITO_APP_CLIENT_ID")
	}

	sess := session.Must(session.NewSession())

	cognitoService := cognitoidentityprovider.New(sess)
	dynamoService := dynamodb.New(sess)

	cognitoInput := cognitoidentityprovider.SignUpInput{
		ClientId: &cognitoAppClientID,
		Username: &evt.Username,
		Password: &evt.Password,
	}

	cognitoResponse, err := cognitoService.SignUp(&cognitoInput)
	if err != nil {
		return Response{Message: "Error creating cognito user: "}, err
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
		return Response{Message: "Error creating dynamo user entry: "}, err
	}

	return Response{Message: fmt.Sprintf("Successfully created user %s", evt.Username)}, nil
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
