---
type: design
audience: backend-dev
status: current
authoritative-source: server/internal/pkg/storage/
last-verified: 2026-04-19
---

# 存储驱动架构

## 目标

定义 `storage` 模块的职责：统一挂载点、驱动注册、能力声明、诊断、健康检查和对象访问接口。

## 边界

`storage` 负责：

- `mount` 模型
- driver 注册
- 能力声明
- 健康检查
- 对象写入 / 查询 / 删除 / 下载链接
- 类型化错误

`storage` 不负责：

- 业务标签与资源绑定
- 课程或社区中的资源展示语义
- 独立维护对象引用表；对象归属由各业务表自行保存

## 当前驱动

- `s3`（覆盖 MinIO / S3 兼容对象存储）

## 预留驱动族

- OpenList 类
- WebDAV
- 私有网盘
- `bhpan`

当前阶段只预留接口与注册入口，不实现真实驱动。

## 核心对象

- `storage_mounts`

## 能力模型

驱动必须显式声明是否支持：

- `put`
- `delete`
- `stat`
- `presigned_download`

不允许通过 panic 或模糊报错表达“不支持”。

## 对象 key 语义

- 业务层与数据库只保存挂载点内的相对 `objectKey`
- 驱动层在访问对象存储时再拼接 `mount.basePath`
- `basePath` 只影响底层对象存储路径，不应回写进业务表

## 错误模型

统一区分：

- 配置错误
- 认证失败
- 权限不足
- 对象不存在
- 网络异常
- 能力不支持
