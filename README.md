# go-grpc-geohashing-service-sample

## The Plan

Prerequisites: Public HostedZone + VPC (in my case a mini VPC due to costs)

* Go
* gRPC
* PostGis or Dynamo
* GitHub Actions
* AWS Infrastructure:
    * Cognito User Group for Auth
    * Public ALB + WAF
    * ECS Fargate Cluster
    * ECR Container Registry
