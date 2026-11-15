# 对外展示（showcase）对接契约

> 目标读者：要在自己的网站 / QRZ 简介 / 博客上接入本站 QSL 展示数据的人。
> 所有响应体和响应头都是实测抓包原样，可直接按此写死对接代码。
> 机制实现在 `backend/internal/plugins/showcase.go`；插件机制说明见 [plugins.md](plugins.md)。

## 环境要求（对接前先确认两项）

1. 后台「插件 → 对外展示」已启用 —— 未启用时 JSON 和徽章都是 404，响应体 `code` 为 `QSL-PLUGIN-OFF`。这个 404 **带** `Access-Control-Allow-Origin: *` 和 `Cache-Control: no-store`，跨域 `fetch` 能读到状态码，也不会被缓存成「一直未启用」
2. CORS 已放行：`Access-Control-Allow-Origin: *`（仅这两个公开端点，以及它们的 `OPTIONS` 预检）。响应里不会同时出现 `Access-Control-Allow-Credentials`。另外带 `Cross-Origin-Resource-Policy: cross-origin`，`<img>` 可以嵌到别人的页面上

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
| `station.*` | 是 | 都可能为空串，别直接当标题用，做回退。`call_sign` 已转成大写，和控制字符、双向文字覆盖符一起清掉 |
| `counts.*` | 是 | int64。`qso` = 通联行数；`cards_sent` = `card_sent` 为真；`cards_signed` = `receipt_confirmed` 为真（与首页公开统计同一口径）；`cards_received` = 收卡记录行数。最后一项**不是** `card_received`（那个表示对方已收到我们寄出的卡） |
| `recent[]` | 是 | 数组可能为空；按日历日、补零后的 `HH:mm`、再按 id 倒序（`8:05` 不会排到 `12:30` 前面）。`date` `YYYY-MM-DD`、`time` `HH:mm`。库存的是 UTC+8 墙钟（和通联列表同一串），这里只把认得出的紧凑日期、带秒的时刻收成这个格式；`rst_*` 可空；对方呼号同样转成大写 |
| `cards[]` | 是 | 可为空；`image_path` **可能为空串**（没传图，或路径不是本站上传文件）。有值时只会是 `/uploads/<文件名>`，拼到站点域名后引用。绝对地址会先收成这个路径；带 `..`、子目录、盘符的值变成空串 |
| `data.recent` 长度 | — | ≤ 快照里的 20 条上限，受 `limit` 截断 |

**必须处理的非 200**：

| 状态 | body.code | 含义 | 建议处理 |
|---|---|---|---|
| 404 | `QSL-PLUGIN-OFF` | 插件被禁用 | 显示「未开启」占位，**不要重试**。跨域也能读到这个状态码 |
| 503 | `QSL-503` | 数据库暂时读失败，且没有上一份可用快照 | 显示失败占位，可退避重试。徽章此时也是这段 JSON，**不是**一张数字全 0 的 SVG |
| 5xx | 其他 | 服务端故障 | 显示失败占位，可退避重试 |

数据库抖动但内存里还有上一份快照时，仍返回 200 和那一份，避免徽章在故障瞬间变成 0。

**缓存与频率**：成功响应头仍是 `Cache-Control: public, max-age=60`。服务端快照默认也留 60 秒，但通联、卡片、收卡、本台资料一写入就会作废，所以绕过浏览器缓存的请求能马上看到新数字。已经贴在 QRZ / 博客上的 `<img>` 仍可能沿用浏览器里那一份，最多再旧 60 秒。客户端轮询间隔 ≥ 60s 即可。`OPTIONS` 预检为 204，`Access-Control-Allow-Methods: GET, OPTIONS`。

## 端点 2：SVG 徽章

```
GET https://你的域名/api/public/ext/showcase/badge.svg
```

- 响应 `Content-Type: image/svg+xml; charset=utf-8`，固定 480×132
- 直接 `<img src>` 引用。响应带 `Cross-Origin-Resource-Policy: cross-origin`，嵌到别人的页面上不会被浏览器拦住
- 缓存同上 `max-age=60`；没有可用快照时是 503 JSON，不是全 0 的图
- 文字内容只有呼号与数字（已 XML 转义，控制字符已去掉），颜色写死不随主题

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
