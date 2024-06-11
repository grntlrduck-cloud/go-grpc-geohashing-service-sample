# go-grpc-geohashing-service-sample

## The Plan

Idea 1:
* Go 
* DynamoDb - GeoHashing
* GitHub Actions
* AWS Infrastructure using CDK Typescript
    * API Gateway 
    * WAF
    * Cognito User Group
    * AWS Lambda: Authorizer Lambda + Application Lambda
    * DynamoDB Table
    * CloudWatch Alerts (using cdk-watchful)

Idea 2:
* Go
* gRPC
* PostGis
* GitHub Actions
* AWS Infrastructure As plantUML markups since the costs for VPC + HostedZone + Domain are a lot of money for a private project