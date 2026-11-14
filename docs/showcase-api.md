# 对外展示（showcase）对接契约

> 目标读者：要在自己的网站 / QRZ 简介 / 博客上接入本站 QSL 展示数据的人。
> 所有响应体和响应头都是实测抓包原样，可直接按此写死对接代码。
> 机制实现在 `backend/internal/plugins/showcase.go`；插件机制说明见 [plugins.md](plugins.md)。

## 环境要求（对接前先确认两项）

1. 后台「插件 → 对外展示」已启用 —— 未启用时所有端点 404
2. CORS 已放行：`Access-Control-Allow-Origin: *`（仅这两个公开端点）

## 端点 1：JSON 数据源

```
GET https://你的域名/api/public/ext/showcase?limit=8
```

| 参数 | 类型 | 范围 | 说明 |
|---|---|---|---|
| `limit` | int | 1–20 | 最近通联条数，默认 8，越界/非法回落 8 |

**响应 200**（实测原样）：

```json
{
  "code": "QSL-0000",
  "message": "成功",
  "data": {
    "station": {
      "call_sign": "BA1ABC",
      "name": "示例电台",
      "name_en": "Demo Radio"
    },
    "counts": {
      "qso": 2,
      "cards_sent": 0,
      "cards_signed": 0,
      "cards_received": 0
    },
    "recent": [
      {
        "call_sign": "BG2ABC",
        "date": "2026-11-10",
        "time": "08:15",
        "band": "40m",
        "mode": "FT8",
        "freq": "7.074",
        "rst_sent": "-10",
        "rst_rcvd": "-08"
      }
    ],
    "cards": [
      { "card_version": "2026 常规卡", "image_path": "/uploads/ab12cd34.png" }
    ]
  }
}
```

字段约定：

| 字段 | 必有 | 说明 |
|---|---|---|
| `code` | 是 | `QSL-0000` = 成功 |
| `station.*` | 是 | 都可能为空串，别直接当标题用，做回退 |
| `counts.*` | 是 | int64，快照口径见下 |
| `recent[]` | 是 | 数组可能为空；`date` `YYYY-MM-DD`、`time` `HH:mm`，均为 UTC+8 墙钟；`rst_*` 可空 |
| `cards[]` | 是 | 可为空；`image_path` **可能为空串**（未传图的卡片版本），且只会是 `/uploads/...` 形式，拼到站点域名后引用；不含 `..`，已过白名单 |
| `data.recent` 长度 | — | ≤ 快照里的 20 条上限，受 `limit` 截断 |

**必须处理的非 200**：

| 状态 | body.code | 含义 | 建议处理 |
|---|---|---|---|
| 404 | `QSL-PLUGIN-OFF` | 插件被禁用 | 显示「未开启」占位，**不要重试** |
| 5xx | 其他 | 服务端故障 | 显示失败占位，可退避重试 |

**缓存与频率**：响应头 `Cache-Control: public, max-age=60`；服务端数据本身 60 秒快照。客户端轮询间隔 ≥ 60s，再快也只是拿到同一份数据。

## 端点 2：SVG 徽章

```
GET https://你的域名/api/public/ext/showcase/badge.svg
```

- 响应 `Content-Type: image/svg+xml; charset=utf-8`，固定 480×132
- 直接 `<img src>` 引用，无需 CORS 考虑（`<img>` 不受跨域读限制）
- 缓存同上 `max-age=60`
- 文字内容只有呼号与数字（已 XML 转义），颜色写死不随主题

## 端点 3：现成嵌入代码（仅站长用，要登录）

```
GET https://你的域名/api/ext/showcase/embed
Authorization: Bearer <登录令牌>
```

实测响应：

```json
{
  "code": "QSL-0000",
  "message": "成功",
  "data": {
    "site_url": "https://qsl.example.com",
    "page_url": "https://qsl.example.com/showcase",
    "feed_url": "https://qsl.example.com/api/public/ext/showcase",
    "badge_url": "https://qsl.example.com/api/public/ext/showcase/badge.svg",
    "html": "<a href=\"https://qsl.example.com/showcase\" target=\"_blank\" rel=\"noopener\"><img src=\"https://qsl.example.com/api/public/ext/showcase/badge.svg\" width=\"480\" height=\"132\" alt=\"QSL QSL\"></a>",
    "https": true
  }
}
```

注意：站点地址没配置时 `site_url/page_url/feed_url/badge_url/html` 全是空串、`https: false`——对接方要先判 `site_url` 非空。`https: false` 时 QRZ 等页面会丢图，需先去设置页把站点地址改成 https。

## 调用方红线

- 只走 GET，无任何写端点，不涉及鉴权（除 embed）
- 不要用 `<iframe>` 嵌 `/showcase` 页面（`X-Frame-Options: DENY` + `frame-ancestors 'none'`，必拒）
- 不要对端点做高频轮询——公开接口有全站 600 次/分钟限速（429 带 `Retry-After`）
- 引用 `image_path` 时用站点地址拼前缀，别引用相对路径（会打到对接方自己的域名上 404）

## 最小可用对接种（可直接抄）

```js
const BASE = 'https://你的域名'

async function loadShowcase(limit = 8) {
  const res = await fetch(`${BASE}/api/public/ext/showcase?limit=${limit}`)
  if (res.status === 404) return { state: 'off' }      // 插件未启用
  if (!res.ok) throw new Error('showcase unavailable')  // 5xx：调用方决定重试策略
  const { data } = await res.json()
  return {
    state: 'ready',
    station: data.station,
    counts: data.counts,
    recent: data.recent,
    cards: data.cards.map(c => ({ ...c, url: c.image_path ? BASE + c.image_path : '' })),
  }
}
```
