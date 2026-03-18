# Casdoor SSO 登录页定制

这个目录存放 Casdoor SSO 登录页使用的自定义 CSS 和 JavaScript，用来把登录页视觉风格对齐到 StuHelper 当前的玻璃拟态设计。

## 文件说明

| 文件               | 用途                                                    |
| ------------------ | ------------------------------------------------------- |
| `custom-login.css` | 登录页主题 CSS，包含青色 / 靛蓝渐变、玻璃卡片和动态光斑 |
| `custom-login.js`  | StuHelper 品牌注入和入场动画脚本                        |

## 当前设计方向

- 背景使用深色渐变，带动态青色 / 靛蓝色光斑
- 登录卡片使用玻璃拟态效果，带模糊、半透明边框和柔和阴影
- 主色与 `stuhelper.com` 对齐，使用青色 `#06b6d4` 和靛蓝 `#4f46e5`
- 主按钮使用渐变背景和悬浮流光效果
- 输入框使用半透明深色玻璃效果和青色聚焦态
- 标题采用系统字体和渐变文字

## 配置步骤

### 1. 进入 Casdoor 管理台

打开你的 Casdoor 管理后台，比如 `https://sso.stuhelper.com`。

### 2. 配置应用级 Custom CSS

1. 进入 **Applications**
2. 选择 StuHelper 对应应用
3. 找到 **Custom CSS**
4. 把 `custom-login.css` 的完整内容复制进去
5. 保存

### 3. 配置应用级 Custom JS

1. 仍然在同一个应用配置页
2. 找到 **Custom JS**，或者版本不同情况下的 **Header HTML** / **Custom Script**
3. 把 `custom-login.js` 的内容复制进去
4. 如果当前版本要求手动包 `<script>`，按下面写：

```html
<script>
	// 在这里粘贴 custom-login.js 的内容
</script>
```

5. 保存

### 4. 配置主题

如果 Casdoor 版本支持应用级主题，建议顺手一起调：

1. 进入 **Applications** > StuHelper > **Theme**
2. 主色设成 `#06b6d4`
3. 如果支持暗色背景，打开暗色模式

这样做的目的是让没有被自定义 CSS 覆盖到的控件也尽量保持一致。

### 5. 配置登录页 Logo

1. 在应用设置里上传 StuHelper Logo
2. 建议使用浅色版本，适配深色背景
3. 建议高度 48px，格式优先 SVG，其次透明底 PNG

### 6. 验证效果

1. 开一个无痕窗口
2. 打开你的登录入口，比如 `https://your-app.stuhelper.com/login`
3. 跳转到 Casdoor 后，确认下面这些点：

- 深色渐变背景和动态光斑存在
- 登录卡片是玻璃拟态效果
- 主按钮是青色到靛蓝的渐变
- StuHelper 标题有渐变字效果
- 页面入场动画正常
- 移动端布局没有坏

## 后续调整说明

### 改颜色

当前 CSS 里主要颜色值是：

- 青色：`#06b6d4`
- 靛蓝：`#4f46e5`
- 深色背景：`#0f172a` / `#1e293b`

要换主题，直接全局搜索这些值再替换。

### Casdoor 版本兼容

这套 CSS 主要针对 Casdoor 默认的 Ant Design 界面结构，选择器覆盖了：

- Ant Design 组件，比如 `.ant-*`
- Casdoor 自带结构，比如 `.login-form`、`.panel-module`
- 一些兜底选择器

如果 Casdoor 后面升级了 DOM 结构，可能要重新调整部分选择器。

### 视觉过渡

如果想让主站跳转到 Casdoor 时更顺，可以这样做：

1. 在主站发起跳转前先显示全屏 loading
2. Casdoor 侧通过 `custom-login.js` 做淡入动画
3. 回调完成后，前端继续保持 loading，直到认证流程结束

这套处理当前已经在 Web 端的 `LoginPage.vue` 和 `AuthCallbackPage.vue` 里落了基础版本。
