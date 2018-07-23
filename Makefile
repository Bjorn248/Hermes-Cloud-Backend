build:
	go get github.com/aws/aws-lambda-go/lambda
	env GOOS=linux go build -ldflags="-s -w" -o bin/user_registration user_registration/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/device_registration device_registration/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/device_update device_update/main.go
