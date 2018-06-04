package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
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

// HandleRequest is the lambda function handler
// it processes the creation of the cognito user
func HandleRequest(ctx context.Context, evt UserRegEvent) (Response, error) {
	var cognitoAppClientID string
	if os.Getenv("COGNITO_APP_CLIENT_ID") == "" {
		log.Fatal("COGNITO_APP_CLIENT_ID not set")
	} else {
		cognitoAppClientID = os.Getenv("COGNITO_APP_CLIENT_ID")
	}

	sess := session.Must(session.NewSession())

	cognitoService := cognitoidentityprovider.New(sess)

	cognitoInput := cognitoidentityprovider.SignUpInput{
		ClientId: &cognitoAppClientID,
		Username: &evt.Username,
		Password: &evt.Password,
	}

	cognitoResponse, err := cognitoService.SignUp(&cognitoInput)
	if err != nil {
		return Response{Message: "Error creating user: "}, err
	}
	fmt.Println(cognitoResponse)
	return Response{Message: fmt.Sprintf("Successfully created user %s", evt.Username)}, nil
}

func main() {
	if os.Getenv("AWS_PROFILE") != "" {
		fmt.Printf("Using AWS Profile: %s\n", os.Getenv("AWS_PROFILE"))
	} else {
		fmt.Println("Using AWS Profile: default")
	}

	if os.Getenv("AWS_REGION") == "" {
		log.Fatal("AWS_REGION not set")
	}

	lambda.Start(HandleRequest)
}
