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

## 前端开发

基于 Vue3 / TypeScript / Vite 构建。

```bash
pnpm install

pnpm dev:web   # 启动 apps/web 注册／重设密码开发服务器
pnpm dev:admin # 启动管理端开发服务器
pnpm build     # 构建工作区包和前端
pnpm test      # 运行工作区测试
```

启动时可根据需要连接不同的后端服务：

- `pnpm dev` 或 `pnpm dev:remote`:对接线上后端 API，适合纯前端修改，请勿提交真实注册、验证邮件或密码修改。
- `pnpm dev:local`:对接本地 Docker 后端 API，需修改后端数据时使用。
- `pnpm dev:native`:对接本地原生启动的后端 API，用于前后端联调。

若需要在手机等设备调试，可添加 `--host` 参数启动服务。

## 后端开发

编译项目

```bash
cd apps/api
go mod download
./script/build_jet.sh # 生成 Jet SQL 代码
go build
```

运行单元测试

```bash
go clean -testcache
go test ./... -v -p 4
```

运行集成测试（需要 Docker）

```bash
./script/test_integration.sh -v
```
