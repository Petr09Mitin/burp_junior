# Start from golang base image
FROM golang:1.23.1-alpine3.19 AS build-stage

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git && apk add --no-cache bash && apk add build-base

ENV GOMODCACHE /go/pkg/mod
ENV GOCACHE /go-cache

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
COPY ca.crt /
COPY ca.key /
COPY .env /

RUN go mod download

RUN CGO_ENABLED=0 go build -o /app ./cmd/app/main.go

FROM gcr.io/distroless/base-debian11 AS build-release-stage

COPY --from=build-stage /app /app
COPY --from=build-stage ca.crt .
COPY --from=build-stage ca.key .
COPY --from=build-stage .env .

EXPOSE 8000 8080

# Run the executable
CMD ["/app"]
