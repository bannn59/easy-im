# 调研方法（刷新）

## 扫描

1. `backend/internal/handler/router.go` 路由表  
2. `frontend/src/app/App.tsx` 路由  
3. `backend/migrations/*`  
4. 既有冒烟：auth、conversations ACL、messages idempotent、WS_OK  

## 证据原则

有路径/符号才标 implemented；hub 存在但不等于产品 presence。
