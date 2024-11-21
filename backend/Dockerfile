FROM golang:1.23.0

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Update the database and file paths in sqlite.go and auth_util.go
RUN sed -i 's|../../pkg/db/app.db|/app/pkg/db/app.db|g' pkg/db/sqlite/sqlite.go && \
    sed -i 's|file://../../pkg/db/migrations|file:///app/pkg/db/migrations|g' pkg/db/sqlite/sqlite.go && \
    sed -i 's|../../pkg/db/uploads|/app/pkg/db/uploads|g' pkg/utilities/auth_util.go

# Build the application
WORKDIR /app/cmd/server
RUN go build -o main .

# Create necessary directories
RUN mkdir -p /app/pkg/db/uploads

EXPOSE 8080

CMD ["./main"]