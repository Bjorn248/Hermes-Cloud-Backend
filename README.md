# Hermes-Cloud-Backend
Lambda functions that serve as the backend API to hermes cloud web

# API Documentation
[Swagger Docs](https://app.swaggerhub.com/apis/PBJ/hermes-cloud-backend/0.0.2)

# Using Serverless
In order to deploy the functions with serverless two additional variables need to be passed
- cognito_app_client_id
- cognito_pool_id
- user_pool_arn

An example deploy would look like the following
```
serverless deploy -v --cognito_app_client_id PLACEHOLDER --cognito_pool_id PLACEHOLDER --user_pool_arn PLACEHOLDER
```
