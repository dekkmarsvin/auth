# Auth 认证服务

[![GPL-3.0](https://img.shields.io/github/license/auto-novel/auth)](https://github.com/auto-novel/auth#license)
[![cd-web](https://github.com/auto-novel/auth/actions/workflows/cd-web.yml/badge.svg)](https://github.com/auto-novel/auth/actions/workflows/cd-web.yml)
[![cd-api](https://github.com/auto-novel/auth/actions/workflows/cd-api.yml/badge.svg)](https://github.com/auto-novel/auth/actions/workflows/cd-api.yml)

提供统一登录认证（SSO）服务，支持用户注册、登录、令牌管理和邮箱验证等功能。

## 贡献

请务必在编写代码前阅读[贡献指南](https://github.com/auto-novel/auth/blob/main/CONTRIBUTING.md)，感谢所有为本项目做出贡献的人们！

## 部署

> [!WARNING]
> 注意：本项目并不是为了个人部署设计的，不保证所有功能可用和前向兼容。

```bash
# 1. 克隆仓库
git clone https://github.com/auto-novel/auth.git
cd auth

# 2. 生成环境变量配置
cat > .env << EOF
REFRESH_TOKEN_SECRET=$(openssl rand -base64 48)
ACCESS_TOKEN_SECRET=$(openssl rand -base64 48)
POSTGRES_PASSWORD=$(openssl rand -base64 48)
MAILGUN_DOMAIN=verify.fishhawk.top
MAILGUN_APIKEY=<mailgun_apikey>
EOF

# 3. 启动服务
docker compose up -d
```

启动后，访问 http://localhost:4000 即可。
