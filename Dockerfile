# syntax=docker/dockerfile:1
<<<<<<< HEAD

# Build Go Program
# FROM golang:1.16-alpine AS build

# WORKDIR /go/src/github.com/superc03/carp

# # Install Dependencies
# COPY go.mod ./
# COPY go.sum ./
# RUN go mod download && go mod verify

# # Transfer Source Files
# COPY *.go ./
# COPY handlers/ ./handlers/
# COPY models/ ./models
# COPY static/ ./static
# COPY templates/ ./templates

# RUN go build github.com/superc03/carp -v -o /carp

# Deploy on Lighter Device
=======
>>>>>>> 06b7b12489b27f50d48999784a33f015ab7cd490
FROM gcr.io/distroless/base-debian10

COPY ./carp /carp

EXPOSE 8080

USER nonroot:nonroot

CMD ["/carp"]
