# 里程碑验收

| 里程碑 | 定义 | 状态 | 验证 |
|--------|------|------|------|
| M0 | 可启动 + health | ✅ | healthz 200 |
| M1 | 能登录 | ✅ | register/login/me smoke |
| M2 | 能建会话并打开 | ✅ | two-user list + stranger 404 |
| M3 | HTTP 消息互可见 | ✅ | send + list + idempotent |
| M4 | 在线实时可见 | ✅ | WS_OK message.created |
| M5 | 已读或在线状态产品化 | ⬜ | 未做（P5） |

## 主线任务

| T | 内容 | 状态 |
|---|------|------|
| T1 | DB + errors | ✅ archived |
| T2 | Auth | ✅ |
| T3 | Conversations | ✅ |
| T4 | HTTP messages | ✅ |
| T5 | WS | ✅ |
| T6 | Feature map refresh | 本任务 |

## 回归命令

```bash
cd backend && go test ./...
cd frontend && npm run build
docker compose up -d
# migrate, api with DATABASE_URL + AUTH_JWT_SECRET
# manual: two browsers register, create conv, send, see via WS/poll
```
