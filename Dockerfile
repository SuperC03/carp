# syntax=docker/dockerfile:1
FROM gcr.io/distroless/base-debian10

COPY ./carp /carp

EXPOSE 8080

USER nonroot:nonroot

CMD ["/carp"]
