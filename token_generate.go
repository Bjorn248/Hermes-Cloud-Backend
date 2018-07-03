package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"log"
	"os"
)

// TokenGenEvent defines the request structure of this token creation request
type TokenGenEvent struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response defines the response structure to this token creation request
type Response struct {
	Message string  `json:"Response"`
	Token   *string `json:"Token"`
}

// CreateToken is the lambda function handler
// it processes the creation of the cognito token
func CreateToken(ctx context.Context, evt TokenGenEvent) (Response, error) {
	var cognitoAppClientID string
	if os.Getenv("COGNITO_APP_CLIENT_ID") == "" {
		log.Fatal("COGNITO_APP_CLIENT_ID not set")
	} else {
		cognitoAppClientID = os.Getenv("COGNITO_APP_CLIENT_ID")
	}

	var cognitoUserPoolID string
	if os.Getenv("COGNITO_USER_POOL_ID") == "" {
		log.Fatal("COGNITO_USER_POOL_ID not set")
	} else {
		cognitoUserPoolID = os.Getenv("COGNITO_USER_POOL_ID")
	}

	var cognitoAuthParams map[string]*string
	cognitoAuthParams = make(map[string]*string)
	cognitoAuthParams["USERNAME"] = &evt.Email
	cognitoAuthParams["PASSWORD"] = &evt.Password

	sess := session.Must(session.NewSession())

	cognitoService := cognitoidentityprovider.New(sess)

	cognitoInput := cognitoidentityprovider.AdminInitiateAuthInput{
		ClientId:       &cognitoAppClientID,
		AuthFlow:       aws.String("ADMIN_NO_SRP_AUTH"),
		AuthParameters: cognitoAuthParams,
		UserPoolId:     &cognitoUserPoolID,
	}

	cognitoResponse, err := cognitoService.AdminInitiateAuth(&cognitoInput)
	if err != nil {
		return Response{Message: "Error creating cognito token: "}, err
	}

	return Response{Message: fmt.Sprintf("Successfully created token"), Token: cognitoResponse.AuthenticationResult.IdToken}, nil
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

	lambda.Start(CreateToken)
}
