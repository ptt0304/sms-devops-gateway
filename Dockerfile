# 🐹 Base image dùng Go
FROM golang:1.21-alpine AS builder

# Cài thêm git (nếu cần go get) và chứng chỉ SSL
RUN apk add --no-cache git ca-certificates

# Tạo thư mục làm việc
WORKDIR /app

# Copy toàn bộ source vào container
COPY . .

# Tải dependencies và build binary
RUN go mod tidy && go build -o sms-devops-gateway ./cmd/main.go

# ----------

# 🌐 Tạo image nhỏ gọn chỉ có binary
FROM alpine:latest

# Tạo thư mục log và file log
RUN mkdir -p /log && touch /log/alerts.log

# Cài tzdata để hỗ trợ timezone và chứng chỉ SSL
RUN apk --no-cache add ca-certificates tzdata

# Set timezone UTC+7 (Asia/Ho_Chi_Minh)
ENV TZ=Asia/Ho_Chi_Minh

# Copy binary từ builder
COPY --from=builder /app/sms-devops-gateway /usr/bin/sms-devops-gateway

# Expose cổng mặc định
EXPOSE 8080

# Chạy app
ENTRYPOINT ["/usr/bin/sms-devops-gateway"]