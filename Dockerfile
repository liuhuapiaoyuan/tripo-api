# 第一阶段：构建阶段
FROM golang:1.22.3-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod  ./
COPY go.sum  ./

# 下载依赖
RUN go mod download

# 复制源码文件
COPY . .

# 编译可执行文件
RUN go build -o /main

# 第二阶段：运行阶段
FROM alpine:latest

# 安装必要的证书
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制可执行文件
COPY --from=builder /main .

# 暴露端口
EXPOSE 8000

# 运行可执行文件
CMD ["./main"]
