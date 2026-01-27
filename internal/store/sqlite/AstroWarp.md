---
title: Default module
language_tabs:
  - shell: Shell
  - http: HTTP
  - javascript: JavaScript
  - ruby: Ruby
  - python: Python
  - php: PHP
  - java: Java
  - go: Go
toc_footers: []
includes: []
search: true
code_clipboard: true
highlight_theme: darkula
headingLevel: 2
generator: "@tarslib/widdershins v4.0.30"

---

# Default module

Base URLs:

# Authentication

# 自部署

## GET 当前用户信息与权限

GET /api/me

统一响应（成功/失败都 HTTP 200）
{
  "ok": true,
  "code": "OK",
  "message": "",
  "data": {},
  "meta": { "traceId": "01JXXXX", "ts": 1736890000 }
}

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Cookie|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "user": {
    "id": 1,
    "username": "clanxie",
    "displayName": "Admin",
    "role": "admin"
  },
  "permissions": [
    "device.read",
    "user_group.read"
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## GET 登录接口

GET /api/login

登录接口,接口交互规范
https://alidocs.dingtalk.com/i/nodes/o14dA3GK8g5nLPQlukk2KMznV9ekBD76?doc_type=wiki_doc&iframeQuery=utm_source=portal&utm_medium=portal_recent&rnd=0.008540239900383106

> Body 请求参数

```json
{
  "username": "alice",
  "password": "******",
  "authMethod": "ldap"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{
  "ok": true,
  "code": "OK",
  "message": "",
  "data": {
    "token": "sid_xxxxx"
  },
  "meta": {
    "traceId": "01JXXXX",
    "ts": 1736890000
  }
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 自部署/设备

## GET 获取设备列表

GET /api/devices

获取设备列表

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Cookie|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "id": 11,
      "ddns": "ddnsxxx",
      "connectedTime": 1736890000,
      "upTime": 3600,
      "ip": "10.0.0.2",
      "mac": "94:83:c4:xx:xx:xx",
      "deviceGroupId": 12,
      "deviceGroupName": "Lab"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 100
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 将设备移动至设备组（批量）

POST /api/devices/move-to-device-group

将设备移动至设备组（批量）

> Body 请求参数

```json
{
  "deviceIds": [
    1,
    2
  ],
  "groupId": 2
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## GET 获取设备组列表（用于“设备移动至设备组”的下拉）

GET /api/device-groups/options

获取设备组列表（用于“设备移动至设备组”的下拉）

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "groupId": 1,
      "name": "Default"
    },
    {
      "groupId": 2,
      "name": "Lab"
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 自部署/设备组

## GET 设备组列表（管理页）

GET /api/device-groups

设备组列表（管理页）

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "id": 2,
      "name": "Lab",
      "deviceCount": 10,
      "description": "Lab devices",
      "userGroupList": [
        {
          "userGroupId": 7,
          "userGroupName": "QA"
        }
      ]
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 添加设备组

POST /api/device-groups

添加设备组

> Body 请求参数

```json
{
  "name": "Lab",
  "description": "Lab devices",
  "userGroupIds": [
    7,
    8
  ],
  "deviceIds": [
    1,
    2
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## GET 用户组列表（用于“编辑设备组”时选择授权用户组）

GET /api/user-groups/options

用户组列表（用于“编辑设备组”时选择授权用户组）

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "userGroupId": 7,
      "name": "QA"
    },
    {
      "userGroupId": 8,
      "name": "Ops"
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## PUT 编辑设备组

PUT /api/device-groups/{id}

编辑设备组

> Body 请求参数

```json
{
  "name": "Lab",
  "description": "Updated",
  "userGroupIds": [
    7
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## DELETE 删除设备组

DELETE /api/device-groups/{id}

删除设备组

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## DELETE 从设备组删除设备（批量）

DELETE /api/device-groups/{id}/devices

从设备组删除设备（批量）

> Body 请求参数

```json
{
  "deviceIds": [
    1,
    2
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 添加设备到设备组（批量）

POST /api/device-groups/{id}/devices

添加设备到设备组（批量）

> Body 请求参数

```json
{
  "deviceIds": [
    1,
    2
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 自部署/用户

## GET 用户列表

GET /api/users

用户列表

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "id": 1,
      "username": "clanxie",
      "role": "admin",
      "userGroupList": [
        {
          "userGroupId": 7,
          "userGroupName": "QA"
        }
      ]
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 新建用户

POST /api/users

新建用户

> Body 请求参数

```json
{
  "role": "user",
  "username": "alice",
  "description": "QA member",
  "password": "xxx",
  "repassword": "xxx",
  "userGroupIds": [
    7
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## PUT 编辑用户

PUT /api/users/{id}

> Body 请求参数

```json
{
  "role": "user",
  "username": "alice",
  "description": "updated",
  "password": "newpass",
  "repassword": "newpass",
  "userGroupIds": [
    7,
    8
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## DELETE 删除用户

DELETE /api/users/{id}

删除用户

> Body 请求参数

```json
{}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 自部署/用户组

## GET 用户组列表

GET /api/user-groups

用户组列表

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{
  "items": [
    {
      "id": 7,
      "userGroup": "QA",
      "description": "QA team",
      "userCount": 5,
      "deviceGroupList": [
        {
          "deviceGroupId": 2,
          "deviceGroupName": "Lab"
        }
      ]
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 添加用户组

POST /api/user-groups

添加用户组

> Body 请求参数

```json
{
  "name": "QA",
  "description": "QA team"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## PUT 编辑用户组

PUT /api/user-groups/{id}

编辑用户组

> Body 请求参数

```json
{
  "name": "QA",
  "description": "updated"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## GET 删除用户组

GET /api/user-groups/{id}

删除用户组

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## PUT 设置用户组可访问的设备组（覆盖式）

PUT /api/user-groups/{id}/device-groups

设置用户组可访问的设备组（覆盖式）

> Body 请求参数

```json
{
  "deviceGroupIds": [
    2,
    3,
    4
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|id|path|string| 是 |none|
|Token|header|string| 否 |none|
|body|body|object| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 数据模型

