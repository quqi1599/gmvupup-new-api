# Responses 首事件前回包诊断

针对流式 Responses 请求在首个有效事件前结束、但现有日志只能看到 `eof/received=0` 的情况，新增管理员日志字段（随对应失败/部分消费记录保存）：

`other.admin_info.upstream_response`

本次只补观察，不改解析、错误码、HTTP 状态、渠道选择、重试、终态或计费。147 是 NewAPI 直连供应商的 OpenAI 类型渠道，不经过本机的 CPA 服务，因此观察点在 NewAPI 的 Responses 读取器。

2026-11-02 用户在 CPA-new 观察增强发布后进一步明确“这次也发布 NewAPI 诊断补丁”。本次恢复这份独立补丁的发布范围，保留 `71c30b05a` 已发布修复；CPA-new 的逐次 HTTP 观察与此处直连 Responses 首事件前观察分别保留。发布按固定提交的自建 CI、签名镜像、生产版本/配置和业务结果核验，不以本文的本机结果替代。

## 字段

| 字段 | 含义 |
|---|---|
| `status_code` | 进入流读取器时的上游 HTTP 状态 |
| `content_type` | 允许列表内的媒体类型；参数不保存，未知值变成 `other`，未提供为 `missing`，`application/*+json` 归为 `application/json` |
| `content_length` | HTTP 客户端给出的长度，`-1` 表示未知；不据此判断响应完整 |
| `read_bytes` | 读取器实际读取的响应体字节，可能已自动解压，并非网络传输字节或已输出字节 |
| `stream_expected` | 适配器读取本次响应时是否按流处理 |
| `request_stream` | 最终编码 JSON 经参数覆盖后的 stream 字段：`true/false/omitted/null/non_boolean`；原样透传等未观察场景为 `not_observed`，不额外读取请求体 |
| `body_prefix` | 前256字节的形状：`empty/whitespace/json_like/html_like/markup_like/sse_like/done_marker/other`；只是前缀特征，不表示整段 JSON 合法或响应完整 |
| `utf8_bom` | 前缀首个非空白位置是否存在 UTF-8 BOM；不自动删除或修复 |
| `data_lines/comment_lines/event_lines/other_lines/blank_lines` | 读取到的物理行类型计数；不保存行内容。`data_lines` 不等于有效事件数。带 BOM 的 data 行仍按当前解析器行为归其他行 |
| `end_reason/soft_errors` | 现有流结束原因与软错误数量，不包含错误正文 |

只有 Responses 的首事件前异常才附加此对象；正常流和已经收到有效事件的流不增加该字段。每个上游尝试/模型轮次重置，防止前一轮诊断污染后一轮。非管理员查询沿用 `admin_info` 整体移除规则。

## 使用边界

- `empty + read_bytes=0 + eof` 是未读到响应体字节。
- `json_like/html_like + data_lines=0` 是非 SSE 前缀线索，不等于已经确定供应商错误原因。
- `comment_lines>0 + data_lines=0` 可识别只观察到心跳/注释的情况；其他行和读取终态仍需一起看。
- `sse_like + utf8_bom=true + data_lines=0` 可暴露 BOM/前缀兼容问题；本次不改变解析结果。
- 原 `relay_stage.body_complete` 只指客户端请求体读完，不能当成上游响应完整。

最多暂存256个响应前缀字节用于分类，不写入上下文、日志或数据库；快照只含枚举、数字和布尔值。不会保存任意响应头、请求/响应正文、工具参数、Cookie、Key 或错误消息，不开启 DEBUG、不额外读完响应体、不自动重放。

本机验证包含真实适配器的空体、JSON、HTML、心跳、BOM、坏JSON帧、缺type帧及正常终态；控制器集成验证诊断落库、HTTP502、单次调用、零消费日志和禁止备用重放。上线后仍需用自然失败请求读取此字段，才可判断147的实际回包类别。

## 2026-11-02 本机验证

- 基于 `71c30b05a`，分支 `codex/stream-response-diagnostics-20261102`；本轮未提交、推送或发布。
- `go test ./relay/helper ./relay/channel/openai ./relay ./controller ./service ./model ./types ./common -count=1 -timeout=240s` 通过；控制器最终夹具调整后全包再次通过。
- 新诊断、显式false、正常终态、客户端取消/超时/读取异常、禁止不安全重放的专项 race 通过；控制器落库和最终stream覆盖验证重复3轮通过。
- 真实HTTP测试确认参数覆盖后的 `stream:false` 发送给模拟上游，诊断同时记录 `request_stream=false / stream_expected=true`，保持原错误及不重放行为；并未把此合成场景当作147的已确认原因。
- 首轮新增落库测试与异步用户设置缓存回填发生清理竞争；夹具改为同步预热完整测试用户后race通过，未修改生产缓存逻辑。
- `go vet` 的 common/custom-event.go 四处复制Mutex警告在未修改的71c30b05a基线同样复现；本轮涉及的其余包vet通过。环境Go1.27.3/darwin-arm64。`git diff --check` 通过。
