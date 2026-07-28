# 消息与实时

| 功能 | status | entry | code |
|------|--------|-------|------|
| 发送消息 | implemented | POST `.../messages`, composer | `message_service.go` |
| 历史 | implemented | GET `.../messages` | `MessageRepo.List` |
| 幂等 | implemented | `client_msg_id` unique | migration + Insert |
| seq 序 | implemented | `conversations.next_seq` | Insert tx |
| WS 推送 | implemented | `/v1/ws`, `message.created` | `hub`, `realtime/index.ts` |
| 已读/输入中 | not_found | — | — |

deps: `messages` 表, hub  
risk: high on fanout path (contained to single node)
