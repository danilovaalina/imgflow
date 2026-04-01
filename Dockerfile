# Стейдж сборки
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Принимаем аргумент, какой именно main собирать
ARG SERVICE_NAME
RUN go build -o /app/service cmd/${SERVICE_NAME}/main.go

# Стейдж запуска
FROM alpine:latest
WORKDIR /root/
# Копируем бинарник и конфиг
COPY --from=builder /app/service .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/web ./web

# Нужно для работы с изображениями (библиотеки декодеров)
RUN apk add --no-cache ca-certificates
CMD ["./service"]
