# 会话创建与列表

## Goal

路线图 **T3 / P2 / M2**：已登录用户可创建会话、查看列表、打开空聊天页；成员 ACL。依赖 T2。不含消息。

## Acceptance Criteria

- [x] 迁移 conversations + members 可 up
- [x] POST/GET conversations + GET by id with ACL
- [x] 非成员不可读详情；未登录 401
- [x] FE workspace 列表/创建/空页
- [x] go test + npm run build
- [x] 无消息功能伪装

## Dependencies

T2 done. Blocks T4.
