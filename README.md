# Project Information

**Project Name**: go-shopping-poc   

**Description**: This project is being used as a proof-of-concept to learn the Go programming language and related concepts.  We are building the microservices needed to support a fictitious shopping application: customer, shoppingcart, order, product, etc.

To support the application we will be using Keycloak (OIDC authentication and authorization), Postgres (database), Kafka (event management), and Minio (S3 storage for product images).  

The microservices and the supporting services will all run inside of a local kubernetes instance from Rancher Desktop running on a a local Mac development machine. 

The front-end application that accesses these services is being written in Angular and is housed in a separate project.  

We will be using the Saga and Outbox patterns to ensure the microservices remain independent. 
