# syntax=docker/dockerfile:1

# Build Go Program
FROM golang:1.16-alpine AS build

WORKDIR /app

# Install Dependencies
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Transfer Source Files
COPY *.go ./
COPY handlers/ ./handlers/
COPY models/ ./models
COPY static/ ./static
COPY templates/ ./templates

RUN go install
RUN go build -o /carp

# Deploy on Lighter Device
FROM gcr.io/distroless/base-debian10

COPY --from=build /carp /carp

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/carp"]