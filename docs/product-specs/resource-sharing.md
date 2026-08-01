---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-07-31
---

# 资源共享

## 目标

定义 `resource` 业务模块。该模块提供资源元数据、标签、绑定、检索和下载入口，用于课程、学期、学校和公共空间中的资源共享。

## 边界

### 包含

- 资源条目
- 资源版本
- 资源标签
- 资源绑定
- 上传、列表、详情、更新、删除、下载链接

### 不包含

- 实验附件
- 作业附件
- 提交物与批改反馈附件
- 面向不存在业务的通用文件中心

## 核心对象

- `resource_items`
- `resource_versions`
- `resource_bindings`
- `resource_tags`

## 业务约束

- 资源本体是业务元数据，不是底层对象 key
- 资源与存储对象解耦
- 版本优先采用 `resource item + version`
- 绑定通过业务语义表示，不把课程、学校等信息硬编码进对象路径

### 上传 MIME 校验

资源创建接口以服务端内容嗅探为安全基线，不直接信任浏览器的 `File.type`：

- 声明类型与嗅探类型相同则接受，并使用嗅探结果。
- ZIP 只接受精确列出的 OOXML、OpenDocument、EPUB、JAR 容器类型和 Windows ZIP
  别名；不接受任意 `vnd.*` 或 `*+zip`。
- OLE 复合文档只有同时具备 OLE 魔数且声明为旧版 Word、Excel 或 PowerPoint 时才接受。
- `text/plain` 只允许细分为 CSV、Markdown、TSV；JSON 还必须通过语法校验。
- 未识别的 `application/octet-stream` 不构成信任任意声明类型的理由；真实矛盾的声明继续
  返回 400，且不会写入对象存储。

通过兼容校验后，资源版本和对象存储记录有效 MIME，而不是一律保存 ZIP、plain text 或
octet-stream 嗅探值。客户端的文件选择提示不是安全边界，不能替代上述服务端校验。

## 最小 API

- 创建资源
- 查询资源列表
- 查询资源详情
- 更新资源元数据
- 删除资源
- 获取下载链接

## 过滤能力

- 关键字
- 标签
- 绑定类型
- 课程
- 学校
- 学期
