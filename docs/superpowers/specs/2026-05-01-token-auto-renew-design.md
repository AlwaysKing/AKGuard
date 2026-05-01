# Token 自动续签设计

## 概述

为 auth token 增加宽限期（grace period）机制。当 token 过期后，在宽限期内用户访问受保护资源被跳转到登录页时，系统自动续签 token 并跳转回原页面，用户无需重新输入密码。

## 流程

```
用户访问受保护资源
    ↓
Nginx auth_request → /verify → token 过期 → 401
    ↓
Nginx 302 → /login?redirect=xxx
    ↓
前端检测 cookie 中有 ak_token？
    ↓ 否                    ↓ 是
正常登录表单          调用 POST /api/auth/renew
                            ↓
                    后端检查：
                    - token 存在
                    - IP 匹配
                    - expires_at + grace_period > now
                            ↓ 通过                ↓ 不通过
                    删除旧 token            返回 401
                    创建新 token            前端走正常登录流程
                    Set-Cookie
                    返回 200
                            ↓
                    前端跳转回 redirect
```

**关键点**：renew 不区分 token 是否已过期。只要没过宽限期，不管 token 还有效还是已过期，都重新生成。唯一拒绝的条件是过了宽限期。

## 后端改动

### 1. 新增配置项

`token_grace_period`：整数，单位秒，默认 `0`（关闭续签功能）。存储在 SQLite settings 表，管理面板可配置。

### 2. 新增接口 `POST /api/auth/renew`

- 从 cookie 取 `ak_token`
- 查找 session，校验 IP
- 判断：`session.expires_at + grace_period > now`
- 通过 → 删除旧 session，创建新 session（同样 IP、UA、token_type），Set-Cookie，返回 200
- 不通过 → 返回 401

### 3. `/verify` 不变

保持严格检查，token 过期直接返回 401。

## 前端改动

### AuthLogin.vue

页面加载时：
1. 检查 cookie 中是否存在 `ak_token`
2. 存在 → 调用 `/api/auth/renew`（credentials: 'include'）
3. 成功 → 跳转回 `redirect` 参数指定的地址
4. 失败 → 正常显示登录表单

## 管理面板改动

在认证方式配置区域新增「Token 宽限期」输入框，单位小时，0 表示关闭。
