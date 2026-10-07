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
MAILGUN_DOMAIN=verify.kotoban.top
MAILGUN_APIKEY=<mailgun_apikey>
# 若使用 EU 區域，設定 MAILGUN_API_BASE=https://api.eu.mailgun.net
TURNSTILE_SITE_KEY=<turnstile_site_key>
TURNSTILE_SECRET=<turnstile_secret>
TURNSTILE_HOSTNAMES=auth.kotoban.top
EOF

# 3. 启动服务
bash script/check_turnstile_config.sh
docker compose up -d
```

启动后，访问 http://localhost:4000 即可。

既有安裝升級前，保留原本 `.env` 中的 token secrets 和 PostgreSQL 資料目錄。
在更新 API 映像前執行 `bash script/apply_db_schema.sh`；腳本先建立並檢查
`backups/` 中的私有 PostgreSQL 備份，再於交易中加入新版管理及處罰提醒需要的
資料表與欄位。舊 `status` 欄位和帳號、密碼、事件、處罰資料均保留。
若存在尚未對應撤銷時間的舊撤銷處罰，升級會停止，需先確認歷史資料的遷移方式。

新版管理介面位於 `https://auth.kotoban.top/admin/`。舊 refresh cookie 可繼續
換發帶 `uid` 的新版 access token；保留 secrets 即可避免重建帳號或重設密碼。

## 第三方服務

Mailgun 使用 `MAILGUN_DOMAIN`、`MAILGUN_APIKEY`；`MAILGUN_API_BASE` 可選，
保留既有區域設定，EU 區域可填 `https://api.eu.mailgun.net`。

Cloudflare Turnstile 保護註冊及找回密碼的驗證碼發送。正式環境需先在 widget
允許 `auth.kotoban.top`，並在既有 `.env` 設定：

| 參數 | 用途 |
| --- | --- |
| `TURNSTILE_SITE_KEY` | 該 widget 的公開站點金鑰 |
| `TURNSTILE_SECRET` | 同一 widget 的伺服器密鑰 |
| `TURNSTILE_HOSTNAMES` | 正式環境固定為 `auth.kotoban.top` |

登入 iframe 在 Auth 網域內執行驗證，因此驗證回傳的 hostname 是
`auth.kotoban.top`。正式環境不得允許 `localhost`、`127.0.0.1` 或測試金鑰。
本 fork 透過 `GET /api/v1/auth/config` 提供公開站點金鑰，無需把它編入 Docker
映像；此回應不快取，且不包含密鑰或 hostname allowlist。

更新服務前執行 `bash script/check_turnstile_config.sh`，確認 Compose 將三個
設定傳入 API。腳本只輸出檢查結果，不顯示金鑰。它檢查本機設定，widget 的
網域及金鑰配對仍需在 Cloudflare 確認。

API 驗證 Siteverify 的 `success`、action（註冊為 `signup`，找回密碼為
`password_reset`）及 hostname 後才執行原本的寄信流程。缺少設定、驗證失敗
或驗證服務無法連線時拒絕請求。既有密碼登入及 cookie 換發不受此驗證影響。
