# 贡献指南

欢迎为本项目贡献！为了确保高效协作，我们有以下建议。

- 如果您计划实现较大的功能或变更，请先通过 Issue 或聊天群联系管理员先行讨论。
- 请保持 Pull Request 的内容精简，聚焦于一个独立的修改点，以便快速检视和合入。

## 以调试模式启动服务

在开发前，建议执行以下命令，以调试模式启动服务。

```bash
export COMPOSE_FILE="docker-compose.yml:docker-compose.debug.yml"
docker compose up -d
```

服务启动后，可通过以下地址访问：
- Web: localhost:4000
- Api: localhost:4000/api
- Postgresql: localhost:4001
- Redis: localhost:4002

## 前端开发

```bash
cd web
pnpm install

pnpm dev        # 默认：连接到线上后端API (同 pnpm dev:remote)
pnpm dev:remote # 连接到线上后端API服务
pnpm dev:local  # 连接到本地Docker Compose后端API服务
pnpm dev:native # 连接到本地启动的后端API服务

pnpm build      # 编译项目
```

## 后端开发

编译项目

```bash
cd api
go mod download
./script/build_jet.sh # 生成 Jet SQL 代码
go build
```

运行集成测试

```bash
go clean -testcache
go test ./... -v -p 4
```
