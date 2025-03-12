FROM golang:1.24.0-alpine3.20
ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev sqlite

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN addgroup -S mercari && adduser -S trainee -G mercari \
    && chown -R trainee:mercari db 

# バイナリをビルド
RUN go build -o server cmd/api/main.go
USER trainee

#`./server` を実行
CMD ["./server"]