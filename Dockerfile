# syntax=docker/dockerfile:1
FROM golang:1.23.5-alpine AS builder

WORKDIR /src

# Install git
RUN apk add git

# Copy go.mod and go.sum files
COPY ./go.mod ./go.sum ./

ARG GH_PERSONAL_ACCESS_TOKEN

RUN git config --global url."https://user:${GH_PERSONAL_ACCESS_TOKEN}@github.com/".insteadOf "https://github.com/"

# Download dependencies
RUN GOPRIVATE=github.com/baupal go mod download

# Copy source code
COPY . .

# Check Go installed
RUN go version

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/app ./cmd

# Final stage
FROM alpine:3.21.2

# https://app.datadoghq.eu/ci/dora
ARG DD_GIT_COMMIT_SHA
ENV DD_GIT_COMMIT_SHA=${DD_GIT_COMMIT_SHA}
ENV DD_VERSION=${DD_GIT_COMMIT_SHA}
LABEL com.datadoghq.tags.version=${DD_GIT_COMMIT_SHA}

# Install bash
RUN apk add bash

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /bin/app /app/

# Expose the port
EXPOSE 8080

# Run the binary
CMD [ "/app/app" ]
