# 会话

| 功能 | status | entry | code |
|------|--------|-------|------|
| 创建会话 | implemented | POST `/v1/conversations`, workspace form | `conversation_service.go` |
| 列表 | implemented | GET `/v1/conversations`, sidebar | `AppShell` |
| 打开会话 | implemented | GET `/v1/conversations/{id}`, `/app/c/:id` | `ConversationRoom` |
| ACL | implemented | 非成员 404 | `GetIfMember` |

deps: `conversations`, `conversation_members`  
risk: low–medium
