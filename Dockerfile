FROM public.ecr.aws/docker/library/golang:1.23.0-alpine3.20 as build_base

ARG APP_NAME=grpc-chagring-location-service

WORKDIR /src

COPY . ./

RUN apk update && apk add --no-cache make openssl

RUN mkdir build

# create self signed sert for communication between app and ALB
RUN openssl req -x509 -nodes -subj "/CN=internal.service.${APP_NAME}" -newkey rsa:4096 -sha256 -keyout build/grpc-key.pem -out build/grpc-cert.pem -days 3650

# download dependencies
RUN make ci

# build the app with build size optimization
RUN go build -ldflags "-s -w" -o build/app cmd/app/main.go
# build our small exetueable for our container health check
RUN go build -ldflags "-s -w" -o build/probe cmd/probe/main.go

FROM scratch

WORKDIR /service

COPY --from=build_base /src/build/ ./
COPY --from=build_base /src/boot.yaml ./
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/service/app"]
