# Start from golang base image
FROM golang:alpine

# Add Maintainer info
LABEL maintainer="jaga"

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git && apk add --no-cach bash && apk add build-base
RUN go install github.com/gobuffalo/pop/v6/soda@latest
# Setup folders
RUN mkdir /app
WORKDIR /app

# Copy the source from the current directory to the working Directory inside the container
COPY . .
COPY config.json .


# Download all the dependencies
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...

# Build the Go app
RUN go build -o /build

# Expose port 8080 to the outside world
EXPOSE 19093

# Run the executable
CMD [ "/build" ]