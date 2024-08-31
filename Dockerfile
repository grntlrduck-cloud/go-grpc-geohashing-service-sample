FROM public.ecr.aws/docker/library/golang:1.23.0-alpine3.20 as build_base

ARG APP_NAME=grpc-chagring-location-service

WORKDIR /src

COPY . ./

RUN apk update && apk add --no-cache make openssl ca-certificates

RUN mkdir -p build/cert

# the certificates created here are only used for TLS between the service and the ALB
# the client traffic TLS context is ALB terminated
# those self signed cert should never be used for anything else, use trusted certs instead!
# create priavte key and ca cert
RUN openssl req -x509 -newkey rsa:4096 -days 3650 -nodes -keyout build/cert/ca-key.pem -out build/cert/ca-cert.pem \
  -subj "/C=DE/ST=Bavaria/L=Munich/O=grntlrduck.cloud/CN=*.grntlr-duck.cloud"
# create server key and cert
RUN openssl req -nodes -newkey rsa:4096 -keyout build/cert/grpc-key.pem -out build/cert/grpc-req.pem \
  -subj "/C=DE/ST=Bavaria/L=Munich/O=grntlrduck.cloud/CN=${APP_NAME}.grntlr-duck.cloud"
# create extras file be able to dial in to local host
RUN echo "subjectAltName=DNS:*grntlr-duck.cloud,DNS:localhost,IP:0.0.0.0,IP:127.0.0.1" > build/cert/extFile.conf
# sign cert with CAs' key
RUN openssl x509 -req -in build/cert/grpc-req.pem -days 3650 -CA build/cert/ca-cert.pem -CAkey build/cert/ca-key.pem -CAcreateserial \
  -out build/cert/grpc-cert.pem -extfile build/cert/extFile.conf

# download dependencies
RUN make ci

# build the app with build size optimization
RUN go build -ldflags "-s -w" -o build/app cmd/app/main.go
# build our small exetueable for our container health check
RUN go build -ldflags "-s -w" -o build/probe cmd/probe/main.go

FROM scratch

WORKDIR /service

# 443 for gRPC and 8443 for the reverseproxy
EXPOSE 443 8443

COPY --from=build_base /src/build/ ./
COPY --from=build_base /src/boot.yaml ./
COPY --from=build_base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/service/app"]
