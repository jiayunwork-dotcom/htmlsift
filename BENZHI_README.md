# htmlsift — Go 语言 HTML 文档解析、链接提取与净化 HTTP 服务

本 HTML 解析与净化 HTTP 服务：读入文档或片段，抽取链接并按 allowlist 剥离危险标签与 URL scheme；净化结果确定且幂等，非法输入须报错。

## 构建 / 运行 / 测试

```text
go mod download        # 首次拉取依赖（此后离线可构建）
go build ./...         # 编译（含 example/）
go test ./...          # run all unit tests
echo '<p onclick="x()">hi</p>' | go run . sanitize -fragment -   # CLI 示例
```

## 评测镜像

本目录评测专用文件（勿覆盖项目自带 Dockerfile/README）：

- `benzhi.Dockerfile`
- `build_benzhi_docker.sh`
- `BENZHI_README.md`（本文件）

两种架构都要构建并进容器验证：

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh <image-name> linux/arm64
./build_benzhi_docker.sh <image-name> linux/amd64
docker run -it <image-name>:latest
```

容器内验证：`cd /app && go build ./... && go test ./...`
