---
type: internal
audience: frontend-dev
status: snapshot
authoritative-source: clients/web + clients/admin + clients/uniappx 源码（develop @ 4a9e30d3）
last-verified: 2026-08-04
---

# 前端与 UI 全面评审报告

> 快照文档。记录 2026-08-04 对 StuHelper 三个前端入口的一次系统性评审结果。
> 结论对应 `develop` 分支 commit `4a9e30d3` 的代码状态；修复推进后本文不再同步更新，以源码为准。

## 1. 评审范围与方法

| 入口 | 代码量 | 技术栈 |
|------|--------|--------|
| `clients/web` | 约 41k LOC，119 个 `.vue` | Vue 3.5 / Vite 6 / TS / Pinia / Tailwind CSS v4 / vue-i18n |
| `clients/admin` | 约 12k LOC | Vben Admin 5（Element Plus 变体） |
| `clients/uniappx` | 约 3.5k LOC，14 个页面 | uni-app x |

方法：

1. **并行深读** —— 17 个评审 agent 分 17 个维度（设计系统、导航 IA、首页与课程发现、搜索与教师页、评课链路、用户中心与通知、认证与长流程、资源与开放平台、管理后台、UniApp X 等）分别深读源码；
2. **对抗性核验** —— 独立的核验 agent **逐条**回源码打假，默认怀疑，专门排查"把已有机制说成缺失"这类误报；
3. **人工复验** —— 报告人对其中 11 条逐行复核，并独立核实了缩进分裂、零引用组件、断点分布、容器宽度散布等量化事实。

产出 **97 条发现，全部经过核验**（另有 1 条经核验驳回，见 §1.2）：

| 核验状态 | 含义 | 条数 |
|------|------|------|
| 已逐行复验 | 报告人亲自打开源码逐行确认 | 11 |
| 部分复验 | 核心事实报告人已确认，附带断言由 agent 核验 | 1 |
| 核验确认 | 核验 agent 回源码确认事实与方案 | 36 |
| 核验调整 | 问题属实，但严重度或方案被修正 | 49 |
| **未核验** | — | **0** |

### 1.1 核验改变了什么

核验不是走过场，它实质性地修正了结论：

- **22 条被下调严重度，0 条被上调。** 原始评审系统性高估严重度。最初报出 14 条 P0，核验后只剩 **5 条**。典型如"入群认证策略页零数据死锁"由 P0 降为 P1（后端仍可建策略，非功能完全不可用）、"全端零安全区适配"由 P0 降为 P2。
- **49 条的修复方案被改写。** 多处原方案本身会引入新 bug。两个例子：
  - "删除 `TeacherFilter.vue` 并清理其 i18n key"——核验发现 `review.filter.all` 并非该组件独占，`CourseDetailPage.vue:578` 正在使用，照原方案删会让课程详情页的教师筛选 chip 显示成裸 key；
  - "院系侧栏错误状态拆分"——原方案要求删掉 `loadDepartments` 里的状态清空行，核验指出这会让 `departmentError` 变成粘滞状态（先失败后成功时错误不消失），应改为改写而非删除。
- **多条事实被证伪。** 如"全站对 `/search` 的引用只有 1 处"——`TeachingHubPage.vue:114` 也有可见入口且已把 `courseName` 写进 URL；如"5 个死代码组件常年参与打包"——`unplugin-vue-components` 只在模板实际用到时注入 import，未被引用的 SFC 不进任何 chunk。

**因此：本文的严重度与修复方案以核验后版本为准。** 每条发现下若有「评审员原方案」折叠块，那是被修正掉的旧版本，仅作存档，不要照着做。

### 1.2 一处自我更正

评审中途曾怀疑 `/courses/list`、`/courses/about`、`/courses/reviews` 三个静态子路由是靠"声明顺序早于 `/courses/:id`"才没被动态段吞掉，属于脆弱设计。

核验 agent 实际调用仓库内的 vue-router 4.6.4 构造了"动态路由先声明、静态路由后声明"的路由表实测，全部正确匹配。**vue-router 4 的 matcher 按 path score 排序插入，静态段（`PathScore.Static`）恒高于动态段（`PathScore.Dynamic`），与声明顺序无关。** 该怀疑不成立，已从结论中剔除。

被驳回的发现：

- ~~`/courses` 静态子路由靠声明顺序才没被 `/courses/:id` 吞掉~~ —— 见上。

---

## 2. 总体判断

**地基比表层好。**

该有的工程基础都在，而且不敷衍：设计 token 体系（`tailwind.css` 的 `@theme` 块）、zh-CN / en-US 双语、可访问性基线（skip link、`useDialogFocus` 焦点陷阱、`useBodyScrollLock`、`eslint-plugin-vuejs-accessibility` 已启用）、路由懒加载 + chunk 加载失败兜底、Storybook、三端 Playwright E2E。

核验过程反复印证了这一点：大量"缺少 XX"的指控被驳回，因为机制其实已经存在——5 个弹窗全都有焦点陷阱和滚动锁，全局 Toast 有统一的模块级状态，路由全部懒加载。

真正的问题是**规范建好了但没有约束力，导致大量代码"写了却根本没生效"**。

这是本次评审最值钱的一类发现：它们在 code review 中看起来完全正常——类名拼得像模像样、变量引用得有板有眼——只有在真实浏览器里才暴露。§3 的 7 条全部属于此类，且**每一条都经过逐行复验**。

三个前端入口的视觉一致性也在恶化：`admission` 模块自带一套 scoped CSS 且被抄了三份，`admin` 的用户系统页硬编码 Tailwind slate 色板，`uniappx` 有 185 处硬编码色值 / 21 种颜色且无任何 token——三者看起来不像同一个产品。

---

## 3. A 类：静默失效（最高优先级）

这七条的共同特征：**代码写了，但浏览器里不生效**。修复总量约 40 行，是整份报告性价比最高的部分。全部经报告人逐行复验。

严重度采用核验后的评级。注意 A2/A3/A4 虽然评为 P1，但它们的**影响面覆盖全站**，实际优先级应高于多数 P1。

### A1 `P0` 所有弹窗被固定顶栏压住

全站 6 处弹窗遮罩用 `z-50`（数值 50），而 `AppHeader.vue:3` 和 `FloatingModuleNav.vue:3` 都用 `z-[var(--z-sticky)]`——`tailwind.css:111` 定义 `--z-sticky: 200`。

`50 < 200`，后果：

1. 遮罩 `bg-black/40` 盖不住顶栏，弹窗顶部被毛玻璃顶栏切掉；
2. 弹窗打开时顶栏的搜索框、通知铃、用户菜单、主题按钮**仍可点击和聚焦**——用户在"确认撤销授权"这类破坏性弹窗上可以直接点走导航，键盘 Tab 会跑到顶栏里；
3. 可拖拽的浮动导航同样浮在弹窗之上。

讽刺的是，`tailwind.css:112-113` 已经定义好了 `--z-modal-backdrop: 300` 和 `--z-modal: 400`，**全站无人使用**。

涉及：`AdminEditDialog.vue:5`、`ModerationDialog.vue:5`、`DraftPromptDialog.vue:5`、`DialogContent.vue:46,52`、`AuthorizedAppsTab.vue:208`、`DeveloperAppsPage.vue:848`。

**改法**：遮罩层换 `z-[var(--z-modal-backdrop)]`，面板换 `z-[var(--z-modal)]`；下拉类（`InlineSearch.vue:37`、`PostReviewPage.vue:96` 的联想列表）换 `z-[var(--z-dropdown)]`。弹窗打开时给顶栏和浮动导航加 `inert`，避免焦点逃逸。补一条 CI 规则禁止 `.vue` 里出现裸 `z-<数字>`。

### A2 `P1` 暗色模式在默认配置下半残

> 评级 P1，但命中的是**默认配置**，实际影响面是全站所有暗色补丁。

- `stores/theme.ts:21`：主题默认值是 `'system'`；
- `stores/theme.ts:41-48` 的 `applyTheme()`：`mode === 'system'` 时执行 `root.removeAttribute('data-theme')`，同时 `root.classList.toggle('dark', isDark.value)`；
- `tailwind.css:5`：`@custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *))` —— **只匹配属性，不匹配 `.dark` 类**。

于是对于"系统偏好暗色 + 从未手动切过主题"的用户（即默认路径）：

- CSS 自定义属性经 `tailwind.css:296` 的 `@media (prefers-color-scheme: dark)` 块正常变暗 ✅
- 全站所有 `dark:` 工具类**全部失效** ❌（如 `AppHeader.vue:3` 的 `dark:border-white/8`、`FloatingModuleNav.vue:20` 的同名类，以及散落各处的十余处暗色补丁）

**改法**：一行。让变体同时匹配 `.dark` 类——store 已经在正确维护这个类了：

```css
@custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *, .dark, .dark *));
```

改完后 `dark-system` 这个自定义变体可以一并删除。

### A3 `P1` 表单错误红框与聚焦边框从不渲染

> 单看 `PostReviewPage` 是 P1；`StudentVerificationPanel` 的同类问题因输入框整个不可见而被核验单列为 **P0**（见 §6「认证与入群长流程」）。

Tailwind v4 的 preflight 对所有元素设 `border-width: 0`。而 `border-danger` / `focus:border-primary` **只设置颜色**，不设置宽度。若元素的基础类里没有独立的 `border` 工具类，这些边框永远不会出现。

`PostReviewPage.vue` 的四个必填字段正是如此——基础类是 `w-full px-4 py-3 bg-bg-elevated rounded-lg ... focus:border-primary`，无 `border`：

| 行号 | 字段 |
|------|------|
| `:69` | 课程搜索 |
| `:199` | 学期 |
| `:287` | 标题 |
| `:327` | 正文 |

后果是双重的：校验错误红框不显示，**`focus:border-primary` 的聚焦边框同样是死的**（只有 `focus:ring-2` 生效）。

`StudentVerificationPanel.vue` 的 10 处输入框（`:138/182/216/242/267/317/335` 及手动认证分支的 `:365/374/394`）情况更严重：`bg-transparent` 且无边框，在 `bg-bg-card` 上是**完全透明的**——只有一个悬空的 placeholder；键盘 Tab 过去也没有任何视觉反馈，违反 WCAG 2.4.7 Focus Visible。

同类问题还出现在 `SearchPage.vue`、`InlineSearch.vue`、`IdentityVerificationPage.vue`。

**改法**：统一对齐 `PhoneBindingPage.vue:90` 的正确写法 `rounded-lg border border-border bg-bg-base/60 ... focus:border-primary focus:ring-2 focus:ring-primary/15`。

### A4 `P1` 未定义的 token 类名渲染成全透明

`bg-bg-input` 在 4 处使用，而 `--color-bg-input` **在 `tailwind.css` 的 `@theme` 里从未定义**。Tailwind v4 因此不生成任何 CSS，元素完全没有背景色：

- `AdminEditDialog.vue:33`（标题输入）
- `AdminEditDialog.vue:46`（正文输入）
- `AdminEditDialog.vue:60`（原因输入）
- `ModerationDialog.vue:34`（审核意见）

核验另确认 `ReplyCard.vue` 存在同类的 `bg-bg-primary`。

**改法**：换成已定义的 `bg-bg-elevated` 或 `bg-bg-base/60`；并加一条 CI grep 规则，拦截 `@theme` 中不存在的 `bg-bg-*` / `text-text-*` / `border-border-*` 类名。

### A5 `P1` `--max-width` 从未定义，顶栏在宽屏无限拉伸

`AppHeader.vue:11` 使用 `max-w-[var(--max-width)]`，但 `--max-width` 在全仓库**没有任何定义**（这是它唯一一次出现）。CSS 变量未定义使该声明在计算值阶段失效，`max-width` 回落到初始值 `none`。

结果：顶栏内容在超宽屏一路铺到边缘，而下方所有页面都被各自的 `max-w-[1200px]` 之类卡住，视觉上顶栏与内容对不齐。

顺带暴露一个系统性问题：全站容器宽度用了 **18 种不同取值**（1200 / 1120 / 1040 / 1000 / 960 / 900 / 800 / 720 / 680 / 640 / 600 / 520 / 420 / 400 / 380 / 360 / 300 / 240 px），103 处 `w-[...]` / `h-[...]` 任意值里只有 4 处引用了 CSS 变量。

**改法**：在 `@theme` 补 `--max-width: 1200px`（或按实际设计定），并把散落的容器宽度收敛成 2–3 个 token。

### A6 `P0` 入群认证的关键 CTA 是原生灰按钮

`AdmissionPage.vue:1084` 是 `<style scoped src="./AdmissionPage.css">`，其中定义了 `.primary-button` / `.secondary-button`。scoped 样式只作用于本组件模板内的元素。

复验结果比最初报告的更精确——**根因是同一套按钮样式被抄了三份，抄漏的那两个就裸了**：

| 组件 | 用了这两个 class | 自己是否也定义了 | 结果 |
|------|------------------|------------------|------|
| `AdmissionPage.vue` | ✅ | 自身模板，scoped 生效 | 正常 |
| `AdmissionReissueHint.vue` | ✅ | ✅ 自己又定义一遍（`:52,69`） | 正常 |
| `OldStudentVerificationFlow.vue` | ✅ | ✅ 自己又定义一遍 | 正常 |
| `FreshmanCameraFlow.vue` | ✅ `:88,97,106` | ❌ style 块里只有 `.field-control` `.field-label` | **失效** |
| `FreshmanMobileCameraPage.vue` | ✅ `:43,57,66` | ❌ 完全没有 `<style>` 块，且是独立路由 | **失效** |

失效的两处渲染成浏览器原生 `button`：无背景色、无圆角、无 44px 最小点击高度、无 disabled 态、无 focus-visible 轮廓。受影响的是"上传材料""回到电脑端继续"等入群认证链路上的关键行动按钮；手机拍照页尤其严重，原生 button 在小屏上只有约 20px 高，主次动作长得一模一样。

**改法**：废弃这套三份重复的 class，统一换用 `components/ui` 的 `<Button>`（`AdmissionPage.vue:346,354` 的确认弹窗已经在用）。若要最小改动，则把 CSS 提升为模块级全局样式并删掉另外两份拷贝。

### A7 `P1` Toast 警告图标是灰色，进出场动画完全不生效

两个独立缺陷叠在同一个组件里：

**① 警告图标渲染成灰色。** `Toast.vue:28` 用 `text-[color:var(--color-rating-3,#f59e0b)]`。但 `--color-rating-3` **确实有定义**（`tailwind.css:68` = `#6b647d`，亮色；`:272` = `#a8a0c0`，暗色）——都是灰色，所以琥珀色 fallback `#f59e0b` 永远不触发。警告 toast 的图标与次要文字同色，视觉上完全不像警告。而正确的 `--color-warning: #e8a840`（`tailwind.css:33`）就在那里没被用。

（成功用 `--color-rating-5` = 绿、错误用 `--color-rating-1` = 红，这两个碰巧是对的；问题在于 rating-3 是"中等评分"的语义色，本就不该拿来表达"警告"。）

**② 进出场动画不存在。** `Toast.vue:60,64` 引用 `fadeInDown` 和 `fadeIn`（驼峰），而 `tailwind.css` 定义的是 `fade-in-down`（`:154`）和 `fade-in`（`:146`，短横线）。名称不匹配 → `animation` 属性引用了不存在的 keyframes → toast 直接闪现闪隐，没有任何过渡。

**改法**：图标色改用 `var(--color-warning)`；动画名改为短横线形式。注意这些 `@keyframes` 定义在 `@theme` 块内，Tailwind v4 仅在对应 `--animate-*` token 被使用时才输出——手写 `animation:` 简写可能仍拿不到，稳妥做法是改用 `animate-fade-in-down` 工具类，或把 keyframes 移出 `@theme` 到普通 `@layer` 中。

---

## 4. B 类：核心转化路径断裂（发布评课）

发布评课是产品最重要的转化路径。这条链路上有四个独立缺陷，**全部经逐行复验**。

### B1 `P0` 提交失败时屏幕上什么都不会变

`PostReviewPage.vue:1018`：

```ts
const course = selectedCourse.value
if (!canSubmit.value || !course) return   // 裸 return
```

不设 `submitError`、不弹 toast、不滚动定位到出错字段。按钮上方那块 `role="alert"` 的告警区（`:399-405`）因此**永远是空的**。

而 `canSubmit`（`:992-1000`）依赖五组条件：课程、学期、全部评分维度、标题、正文。其中：

- **标题**和**正文**有内联文字错误（`#review-title-error` `:298`、`#review-content-error` `:341`）；
- **课程**、**学期**、**评分维度**三项**只有** A3 那个永不渲染的红框作为提示。

提交按钮 `:disabled="submitting"` 只在提交中禁用，表单没填完照样可点。

**净效果**：漏选某个评分维度的用户点击提交，页面毫无反应。他会反复点击，然后放弃。这是整个产品最重要的转化路径上的死结。

**改法**：`if (!canSubmit.value || !course)` 分支里写入 `submitError` 并 toast；`await nextTick()` 后 `document.querySelector('[aria-invalid="true"]')?.scrollIntoView({ block: 'center' })` 并聚焦；错误块里列出未完成项并支持点击跳转。

### B2 `P1` 敏感词提示与实际行为相反

`PostReviewPage.vue:1032` 附近：

```ts
if (!checkResult.isValid) {
  if (checkResult.level === 'warn') {
    toast.warning(t('review.post.contentWarning'))   // 提示后继续往下走
  } else {
    submitError.value = t('review.post.contentBlocked')
    toast.error(submitError.value)
    return                                            // 只有 block 级才拦
  }
}
// ↓ warn 级别直接落到这里，评课照常发出
await api.review.createReview(payload)
toast.success(t('review.post.success'))
```

而 `zh-CN/review.ts:98` 的文案是：**「内容可能含有敏感词汇，请检查后提交」**。

用户看到一个要求他"检查后提交"的警告 toast，紧接着又看到"发布成功"并被跳走。他以为被拦下待修改，实际内容已经公开发布。两个 toast 自相矛盾。

**改法**：二选一——要么 `warn` 级别改为拦截并让用户确认（"仍要发布 / 返回修改"），要么把文案改成陈述事实的"内容已发布，可能含敏感词汇，将进入人工复核"。当前的措辞与行为组合是不可接受的。

### B3 `P1` 恢复草稿弹窗无路可退

`PostReviewPage.vue:421-431` 挂载恢复草稿弹窗时：

- **没有绑定 `@keep`**；
- **没有传 `cancel-text`**。

而 `DraftPromptDialog.vue` 的实现是正确的：`:7` 的 `@click.self="dismiss"` 和 `:85-89` 的 `useDialogFocus({ close: dismiss })` 都会 `emit('keep')`。取消按钮是 `v-if="cancelText"`（`:27`）。

于是：Esc 和点击遮罩都 emit 一个无人监听的事件，弹窗纹丝不动；取消按钮根本不渲染。用户只剩两个选项——「恢复草稿」或「放弃草稿」（不可撤销删除）。想保留草稿又想从头写的用户无路可走。

**缺陷在调用方而非组件**：同一文件 `:433-443` 的离开确认弹窗绑了 `@keep="resolveLeavePrompt(true)"`，行为完全正常。

**改法**：给恢复弹窗补 `@keep` 和 `:cancel-text`。

### B4 `P0` `isOwner` 从未被解析

OpenAPI 契约里有该字段（`clients/shared/src/types/api.gen.ts:3801` 与 `:3860`，注释为"当前已认证用户是否为该测评作者"），但 `clients/web/src/modules/review/reviewListPayload.ts` **整个文件没有出现过 `isOwner`**。

`getLatestReviewsPage` / `getReviewsPage` / `searchReviewsPage` 三个列表接口都走这个 reader，因此 `review.isOwner` 恒为 `undefined`。`ReviewCard.vue:427` 的 `isOwn = props.isOwnReview ?? (review.isOwner === true)` 恒为 `false`——只有 `MyReviewsTab` 显式传 `:is-own-review="true"` 才为真。

**后果**：用户在课程详情页、信息流、搜索结果里看自己刚发的评课，没有编辑和删除入口，**却有"举报"按钮——可以举报自己**。发布成功后正是跳转到 `/courses/:id/reviews`，用户第一眼看到的就是这个状态。

**改法**：在 `readReviewPayload` 里补 `isOwner` 的布尔解析。

---

## 5. C 类：结构性问题（需要产品决策）

以下几条改动面较大、涉及产品形态，列出供决策，不建议在未确认前动手。核验对其中数条的论据做了收窄，此处已采纳。

### C1 三套导航相互重复

顶栏 4 项（首页 / 课程 / 教师 / 资源）、右下角可拖拽浮动导航 3 项（评课 / 教师 / 资源）、首页 3 张功能卡——指向同一批目的地。

`FloatingModuleNav` 在移动端退化成一个固定右下角、语义不明、**无法关闭**的圆钮，会遮挡页面内容与底部操作按钮。桌面端的可拖拽 + 悬停展开对键盘用户不友好。

**建议直接删除该组件**：它没有提供顶栏之外的任何价值，且引入了拖拽状态持久化、视口 clamp、点击抑制等约 290 行复杂度。

### C2 课程列表被三层入口埋掉

`/courses` 是枢纽页（`TeachingHubPage`），`/courses/list` 才是课程列表，`/courses/reviews` 是评课流，`/courses/:id` 是详情。顶栏"课程"指向枢纽页。

全站**只有一个链接**指向 `/courses/list`（`TeachingHubPage.vue:137`）。

**建议**：压平成两层，顶栏"课程"直接进列表，枢纽页的内容（热门课程等）并入列表页顶部或首页。

### C3 账号设置散成 11 个入口

`/user/{reviews,votes,favorites}` 三个路由共用 `UserCenterPage` 的 tab（这个做法是对的），但 `/account/profile`、`/account/security`、`/user/academic-info`、`/user/authorized-apps`、`/user/identity-verification`、`/user/student-verification`、`/user/phone-binding`、`/user/qq-binding` **各自是独立整页**，之间没有任何统一导航。

同时存在三个语义重叠的"首页"：`/identity`（IdentityHomePage）、`/user/reviews`（UserCenterPage + ProfileSection）、`/account/profile`。

**建议**：合并为带左侧栏的 `/settings` 区。

### C4 全局搜索缺可见入口

> 核验收窄：并非"全站无搜索"。`CourseListPage.vue:209-231` 有可见搜索框，`TeachingHubPage.vue:24-48` 有 hero 搜索框且 `:114` 已有「高级搜索」链接。真正的缺口是下面三条。

- `CommandPalette` 在 `AppShell.vue:18` 全局挂载，但 `useCommandPalette` 的 `open()` / `toggle()` **没有任何 UI 调用点**，唯一触发是 `⌘/Ctrl+K`——触屏用户永远打不开，界面上也没有 `⌘K` 提示；
- 顶栏 `InlineSearch` 只在 `route.path === '/courses'` 精确匹配时出现（`AppHeader.vue:236`），课程详情页、教师主页、资源页都没有；
- 顶栏搜索按 Enter 是死路（`InlineSearch.vue:216-222` 要求先用方向键选中），且只取 top-10（`:161`），排名之外的课搜不到也到不了高级搜索页。

### C5 权限不足时静默弹回首页

`router/index.ts:620-628` 在 capability 不满足时直接 `return { name: "home" }`，全局无 `afterEach` 或 toast 兜底。

叠加 `TeachingHubPage.vue:161-170` 的主 CTA「写测评」是**裸 `router-link`**，绕过了 `AppHeader.vue:284-293` 已有的 `ensureCanPostReview` 检查——无权限用户点击枢纽页最显眼的按钮，会被无声弹回首页，得不到任何解释。

---

## 6. 分维度完整清单

全部 97 条发现的完整明细，按维度与严重度排列。每条都标注了核验状态：

- 标注**核验调整**的，「建议改法」已是修正后版本，其下的「评审员原方案」折叠块是被否掉的旧版，**不要照着做**；
- 每条的「核验记录」折叠块记录了核验依据与行号，可据以复核。

### 设计系统与视觉基础

共 8 条：P0 1 / P1 5 / P2 2

#### `P0` 所有弹窗 z-50 都被固定顶栏 z-sticky(200) 压住，模态框顶部被遮挡且底层按钮仍可点

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/styles/tailwind.css`
- `clients/web/src/components/layout/AppHeader.vue`
- `clients/web/src/components/layout/FloatingModuleNav.vue`
- `clients/web/src/components/ui/dialog/DialogContent.vue`
- `clients/web/src/modules/user/views/AuthorizedAppsTab.vue`
- `clients/web/src/modules/open-platform/views/DeveloperAppsPage.vue`

**现状**

tailwind.css:110-115 定义了完整 z 层级：--z-dropdown:100 / --z-sticky:200 / --z-modal-backdrop:300 / --z-modal:400 / --z-toast:500 / --z-tooltip:600。AppHeader.vue:3 的固定顶栏用 z-[var(--z-sticky)]（=200），FloatingModuleNav.vue:3 也是 200。但所有实际弹窗都写死了 Tailwind 的 z-50（=50）：DialogContent.vue:46 和 :52、AuthorizedAppsTab.vue:208（Teleport to body 的撤销授权确认框）、DeveloperAppsPage.vue:848（Teleport to body 的原因填写框）、AdminEditDialog.vue:5、ModerationDialog.vue:5、DraftPromptDialog.vue:5，共 9 处。

**问题**

50 < 200。这些弹窗都是 Teleport 到 body 的 position:fixed inset-0 全屏遮罩，和顶栏处于同一个根层叠上下文，结果就是：①遮罩盖不住顶栏，弹窗顶部 56px（--navbar-height）永远被 backdrop-blur 的毛玻璃顶栏切掉；②居中的弹窗在小屏/内容较高时标题会被顶栏压住；③顶栏里的搜索框、通知铃、用户菜单、主题按钮在弹窗打开状态下仍可点击和聚焦，用户在『确认撤销授权』这种破坏性弹窗上可以直接点走导航，键盘 Tab 也会跑到顶栏里去；④FloatingModuleNav 同样浮在弹窗之上。已经定义好的 --z-modal-backdrop/--z-modal 一次都没被用过（grep 只命中 tailwind.css 自身的定义行）。

**建议改法**

把 9 处 z-50 全部换成层级 token：遮罩层用 z-[var(--z-modal-backdrop)]，弹窗面板用 z-[var(--z-modal)]；下拉类（InlineSearch.vue:37、PostReviewPage.vue:96 的联想列表）换成 z-[var(--z-dropdown)]。同时在弹窗打开时给顶栏和 FloatingModuleNav 加 inert/aria-hidden（或在 body 上加一个 data-modal-open 属性让二者 pointer-events:none），避免焦点逃逸。最后加一条 ESLint/stylelint 或 grep 级 CI 检查，禁止在 .vue 里出现裸 z-<数字>，强制走 --z-* token。

---

#### `P1` Toast 的警告图标渲染成灰色，且进出场动画引用了不存在的 keyframes 名

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/components/common/Toast.vue`
- `clients/web/src/styles/tailwind.css`

**现状**

Toast.vue:16-33 的四种类型图标用的全是评分色阶而不是语义色：success → text-[color:var(--color-rating-5,#10b981)]、error → var(--color-rating-1,#ef4444)、warning → var(--color-rating-3,#f59e0b)。而 tailwind.css:68 定义 --color-rating-3: #6b647d（灰紫），暗色 :272 是 #a8a0c0（灰）。另外 Toast.vue:60/64 的 .toast-enter-active { animation: fadeInDown ... } 和 .toast-leave-active { animation: fadeIn ... } 引用的是驼峰名，而 tailwind.css:146-157 里定义的是 kebab-case 的 fade-in / fade-in-down。

**问题**

①warning toast 的 AlertTriangle 图标解析出 #6b647d——和同一个 toast 里 :36 的关闭按钮 text-text-muted 是**同一个色值**，警告图标和一个次要按钮长得一模一样，完全丧失『黄色=警告』的语义信号。因为 --color-rating-3 是有定义的，CSS var 的 #f59e0b 兜底值永远不会生效，这个 bug 在代码里看不出来。同时 --color-warning:#e8a840 / --color-success:#52c07a / --color-danger:#d86060 这套语义色就摆在那儿没人用。②fadeInDown/fadeIn 在全仓库（tailwind.css、main.css）都不存在——SearchPage.vue:820 里那个 @keyframes fadeIn 在 <style scoped> 内且会被 SFC 编译器改名，够不着 Toast。结果是 toast 完全没有进出场动画，直接闪现闪没，TransitionGroup 白写。

**建议改法**

①图标语义色改法不变：Toast.vue:18/23/28 分别换成 text-success / text-danger / text-warning（--color-success/#52c07a、--color-danger/#d86060、--color-warning/#e8a840 在 tailwind.css:31-35 与暗色 :244-246 两套主题都有定义），去掉 var() 兜底。②动画必须让 Tailwind 真正产出 keyframes——推荐删掉 .toast-enter-active/.toast-leave-active 两条 scoped 规则，改在模板上用 <TransitionGroup enter-active-class="animate-fade-in-down" leave-active-class="...">：使用 animate-fade-in-down 工具类会同时触发 --animate-fade-in-down 与 @keyframes fade-in-down 的产出。离场没有对应 token，可保留一条 scoped .toast-leave-active { animation: fade-in var(--duration-fast) var(--ease-out) reverse; }（fade-in 已被 35 处使用，keyframes 确定在产物里），或改用简单的 opacity/transform transition。若坚持只在 scoped CSS 里写动画名，就必须在同一 SFC 内自带 @keyframes 定义，不能依赖 @theme 里的声明。③role/aria-live 按类型分流的建议可原样保留。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

①把三处图标色改成语义 token：success 用 text-success、error 用 text-danger、warning 用 text-warning，删掉 var() 兜底值（token 两套主题都有定义）。②把 :60/:64 的动画名改成 fade-in-down / fade-in，或者直接删掉 scoped style 改用已有的 --animate-fade-in-down 工具类。③顺带把 :12 的 role="alert" aria-live="assertive" 按类型分流——success/info 用 role="status" aria-live="polite"，只有 error/warning 保留 assertive，避免每条成功提示都打断读屏。

</details>

<details><summary>核验记录</summary>

事实全部核对无误：clients/web/src/components/common/Toast.vue:18/23/28 分别用 text-[color:var(--color-rating-5,#10b981)] / rating-1 / rating-3；clients/web/src/styles/tailwind.css:68 --color-rating-3:#6b647d 与 :52 --color-text-muted:#6b647d 逐字节相同（Toast.vue:38 关闭按钮正是 text-text-muted），暗色 :272 rating-3 #a8a0c0 == :259 text-secondary，因此 var() 兜底 #f59e0b 永不生效，警告图标确实和次要按钮同色。:60/:64 的 fadeInDown / fadeIn 全仓库仅此两处（grep 全仓：仅 Toast.vue:60、Toast.vue:64，另有 modules/review/views/SearchPage.vue:817/820 的同名 keyframes 但在 :815 <style scoped> 内会被 SFC 改名），tailwind.css:146/154 定义的是 fade-in / fade-in-down，动画确实是死的。role="alert" aria-live="assertive" 硬编码在 :11-12 也属实；toast.warning 确有真实调用方（modules/review/views/PostReviewPage.vue:1034）。判 ADJUSTED 的唯一原因是原方案②的首选做法不可行：我用仓库自带 tailwindcss@4.3.3 的 compile() 直接编译 clients/web/src/styles/tailwind.css，喂入代码里实际使用的候选类，输出中真实产出的 @keyframes 只有 spin/pulse/fade-in/fade-in-up/vote-bounce/shake/modal-in/shimmer/aurora-drift/border-rotate —— fade-in-down 的 keyframes 与 --animate-fade-in-down 变量都被 Tailwind v4 摇掉了（全仓没有任何 animate-fade-in-down 用法）。也就是说把 Toast.vue:60 单纯改名成 fade-in-down，编译产物里依然没有这个 keyframes，入场动画照样是死的，改完还会误以为修好了。

</details>

---

#### `P1` dark: 变体在默认的『跟随系统』模式下完全失效，13 处暗色补丁全部不生效

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/styles/tailwind.css`
- `clients/web/src/stores/theme.ts`
- `clients/web/src/modules/user/views/QQBindingPanel.vue`
- `clients/web/src/components/layout/AppHeader.vue`
- `clients/web/src/components/common/NotificationBell.vue`

**现状**

tailwind.css:5 用 @custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *)) 覆盖了 Tailwind v4 内置的 dark 变体，因此 dark: 只在 html 上有 data-theme="dark" 时命中。tailwind.css:8 另外定义了 dark-system 变体专门覆盖系统偏好场景，但全仓库 grep 只命中它自己的定义行，一次都没用过。stores/theme.ts:19-21 默认 mode='system'，theme.ts:44-49 的 applyTheme 在 system 模式下 removeAttribute('data-theme')，然后 root.classList.toggle('dark', isDark.value) 加了个 .dark class——而 .dark 已经不在 dark 变体的选择器里，没有任何样式读它。

**问题**

新用户默认就是 system 模式。当 OS 处于暗色时：CSS 变量走 tailwind.css:296 的 @media(prefers-color-scheme:dark) 块，背景/文字正确变暗；但所有 dark: 工具类静默失效。实测受影响的线上代码有 13 处——10 处 dark:border-white/8（AppHeader.vue:6、NotificationBell.vue:25、AppUserMenu.vue:43、InlineSearch.vue:37 等玻璃面板边框，会保持 white/15，暗色下亮度是设计值的 2.5 倍，边框刺眼），以及 QQBindingPanel.vue:72/101/128 的 text-amber-600 dark:text-amber-400 与 text-amber-700 dark:text-amber-300——暗色下退回 amber-700 (#b45309) 压在 #1e1830 卡片上，实测对比度约 2.6:1，远低于 WCAG AA 4.5:1，QQ 绑定的警告文案基本读不清。更严重的是这是个持续陷阱：后续任何人写 dark: 都会在最常见的默认模式下悄悄失效。

**建议改法**

两条路选一条。推荐 A：把变体定义扩成 @custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *, :root:not([data-theme="light"]):not([data-theme="dark"]) *))，让 dark: 同时覆盖显式暗色和系统暗色，删掉从此多余的 dark-system 变体；同时删掉 theme.ts:48 那行没人消费的 classList.toggle('dark')。B：改 applyTheme，system 模式下把解析结果写成 data-theme=resolvedTheme（另存一个 data-theme-mode 记录用户选择），让 data-theme 永远有值。另外 QQBindingPanel 的 amber-600/700 应改用 text-warning（--color-warning #e8a840，两套主题都有定义），别用 Tailwind 原生色板绕开 token。

---

#### `P1` 两套同名 Button 并存，shadcn 原语硬编码 slate 色板且无暗色，招生页在暗色下白底白框

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/components/ui/Button.vue`
- `clients/web/src/components/ui/button/ShadcnButton.vue`
- `clients/web/src/components/ui/input/Input.vue`
- `clients/web/src/components/business/HeroSection.vue`
- `clients/web/src/modules/admission/views/AdmissionPage.vue`

**现状**

仓库里有两个都叫 Button 的组件。components/ui/Button.vue 走 token（.btn-primary 用 var(--color-primary)，变体 primary/secondary/ghost，尺寸 sm/md/lg），通过 unplugin-vue-components 自动注册（components.d.ts:18），被 HeroSection.vue:38/41 使用。components/ui/button/ShadcnButton.vue 通过 ui/button/index.ts 重命名导出为 Button，被 AdmissionPage.vue:385 显式 import 并在 :346/:354 使用，变体是 default/outline/secondary/destructive/ghost，样式全部硬编码 Tailwind 原生 slate：default 'bg-slate-950 text-white'、outline 'border border-slate-200 bg-white text-slate-900'、destructive 'bg-red-600'、focus ring 'ring-slate-950'，且**一个 dark: 变体都没有**。同目录的 Input.vue:24-26 同病：'border-slate-200 bg-white text-slate-950 placeholder:text-slate-400 focus-visible:ring-slate-950'。

**问题**

①同一个 app 里两种按钮视觉语言：首页 Hero 的主按钮是品牌蓝 #3f5ccb，招生认证弹窗的主按钮是近黑 slate-950，取消按钮是纯白描边——用户在两个页面看到的『主操作按钮』长得不像同一个产品。②变体名还冲突：ui/Button 的 primary/secondary/ghost vs ShadcnButton 的 default/outline/...，写代码时 <Button variant="primary"> 会因为文件里有没有那行 import 而落到完全不同的实现上，TS 也不报错（自动注册没有类型约束）。③暗色下直接坏掉：AdmissionPage 的绑定确认弹窗在 #151020 底色上出现 bg-white/text-slate-900 的取消按钮和 bg-white 的 Input，是刺眼的纯白块；ring-slate-950 的聚焦环在暗背景上等于不可见。

**建议改法**

保持『保留 ShadcnButton API + 全量换 token + 删 ui/Button.vue + 改 HeroSection.vue:38/41 变体名 + 清 components.d.ts:18』的主线不变，但必须把整套 shadcn 原语一起迁移，不能只改 Button/Input：同批修改 ui/dialog/DialogContent.vue:53（'border border-border bg-bg-card p-6 text-text-primary shadow-xl'）与 :66 关闭按钮（text-text-muted hover:bg-bg-hover hover:text-text-primary focus-visible:ring-primary/40）、DialogTitle.vue:17（text-text-primary）、DialogDescription.vue:17（text-text-secondary）；Button 的 outline 变体在弹窗内要保证边界可见，建议 'border border-border bg-transparent text-text-primary hover:bg-bg-hover' 而不是 bg-bg-card（否则亮色下与白色面板同色）；focus ring 统一 focus-visible:ring-primary/40 或 shadow-focus-ring。另外 AdmissionPage.vue:337 的 text-red-600 等零散硬编码色也一并换 text-danger。改完建议加一条 lint/grep 守卫禁止 components/ui/ 下出现 slate-/gray-/red- 原生色板。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

保留一套。建议留 ShadcnButton 的 API（有 disabled、class 透传、size=icon，比 ui/Button.vue 完整），但把 variantClasses/sizeClasses 里的 slate/red 全部换成 token：default 'bg-primary text-text-inverse hover:bg-primary-dark'、outline 'border border-border bg-bg-card text-text-primary hover:bg-bg-hover'、secondary 'bg-bg-secondary text-text-primary'、destructive 'bg-danger text-text-inverse'、ghost 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'、focus ring 换 focus-visible:ring-primary/40（或直接用 shadow-focus-ring）。Input.vue 同样换成 border-border bg-bg-elevated text-text-primary placeholder:text-text-muted。然后删除 components/ui/Button.vue，把 HeroSection.vue:38/41 的 variant="primary"/"secondary" 改成 default/secondary，并从 components.d.ts 清掉重复注册。

</details>

<details><summary>核验记录</summary>

核心事实准确：clients/web/src/components/ui/Button.vue:3 变体 primary/secondary/ghost + :36-49 走 var(--color-primary) token；clients/web/src/components/ui/button/ShadcnButton.vue:25-31 变体 default/outline/secondary/destructive/ghost 全部硬编码 bg-slate-950 / border-slate-200 bg-white text-slate-900 / bg-red-600，:21 focus ring 是 ring-slate-950，全文件 0 个 dark:；clients/web/src/components/ui/input/Input.vue:25-26 同病；button/index.ts:1 把 ShadcnButton 重导出为 Button；components.d.ts:18 确实把 ui/Button.vue 全局注册为 Button（同时 :63 又注册了 ShadcnButton），AdmissionPage.vue:385 显式 import { Button } from '@/components/ui/button' 并在 :346/:354 使用，HeroSection.vue:38/41 用 variant="primary"/"secondary"（细节修正：HeroSection.vue:4 是显式 import '../ui/Button.vue'，不是靠自动注册，但『同名两实现、加不加那行 import 结果完全不同』的结论成立）。我另用 tailwind 4.3.3 实测编译确认 bg-slate-950/bg-white/ring-slate-950 都能正常产出（默认色板未被 @theme 覆盖），dark: 编译为 :where([data-theme="dark"], ...)。判 ADJUSTED 是因为问题③的归因和方案都漏了真正的白块来源：clients/web/src/components/ui/dialog/DialogContent.vue:53 面板本身就是 'border-slate-200 bg-white p-6 text-slate-950'，:66 关闭按钮 text-slate-500/hover:bg-slate-100，DialogTitle.vue:17 text-slate-950、DialogDescription.vue:17 text-slate-600 —— 暗色下刺眼的纯白块是整个弹窗面板，取消按钮/Input 只是待在白面板上而已（ShadcnButton 与 Input 全仓也仅在这一个弹窗里用：grep <Input 只有 AdmissionPage.vue:324，<Button 只有 :346/:354）。若按原方案只把 Button/Input 换成 token 而不动 dialog/*，暗色下会变成『白色面板 + bg-bg-card(#1e1830) 深色按钮 + bg-bg-elevated(#281f3d) 深色输入框 + text-text-primary 近白文字』，亮色下 outline 按钮又会 bg-bg-card(#ffffff) 叠在 bg-white 面板上、边框只剩 rgba(0,0,0,0.06)，比现状更糟。

</details>

---

#### `P1` 文字层级 4 个 token 只落到 2 个真实色值，rating-3 与 text-muted 完全相同

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/styles/tailwind.css`

**现状**

tailwind.css:50-53 亮色：--color-text-primary #2e2845 / --color-text-secondary #635c78 / --color-text-muted #6b647d / --color-text-tertiary #6b647d。muted 和 tertiary 逐字节相同；secondary(#635c78) 与 muted(#6b647d) 每个通道只差 5-8/255。暗色 :260-261 同样 --color-text-muted 和 --color-text-tertiary 都是 #8f86a5。评分色阶 :66-70 是 rating-1 #d86060(红) / rating-2 #d89850(橙) / rating-3 #6b647d / rating-4 #52c07a(绿) / rating-5 #40b090(青绿)，rating-3 恰好等于 --color-text-muted；暗色 :272 的 rating-3 #a8a0c0 恰好等于暗色 --color-text-secondary。

**问题**

①对外承诺 4 级文字层级，实际只有 2 级：写 text-text-secondary 还是 text-text-muted 还是 text-text-tertiary，屏幕上看不出区别，于是全站随机混用（三者都在用），信息层级失效——卡片标题下的元信息和真正的辅助说明是同一个灰度。②rating-3 是整条色阶里唯一的无彩色：1红 2橙 4绿 5青绿构成连续色相渐变，中间突然插一个灰紫，评分 3 分（『中等』）在 RatingBar/RatingCircle/EmojiRating/SemesterStatsGrid（都通过 design-system/rating.ts 取色）上看起来像『未评分/禁用』而不是『中等』，而且它和周围的 muted 正文同色，读者根本不会把它当成一个评分信号。③实测对比度：#6b647d 在 --color-bg-tertiary #dcd8e8 上约 4.0:1，未过 AA；在 --color-bg-base #ece8f4 上 4.65:1，刚过线没有余量。

**建议改法**

①把 4 级压成 3 级并拉开间距：删掉 --color-text-tertiary（全站替换为 text-text-muted），把 --color-text-muted 调深到 #57506b 左右（白底约 7.5:1、bg-tertiary 上约 5.3:1，明显区别于 secondary 又稳过 AA），--color-text-secondary 保持 #635c78 或稍微提亮到 #6f6889 以拉开与 primary 的差距；暗色同步调整（muted 提亮到 #a29ab8 一档）。②rating-3 换成有彩色的中性档，例如 --color-rating-3: #d8b84a（暖黄，接在红-橙-绿之间色相连续）或直接复用 --color-warning #e8a840，暗色用 #f0c060；这样 1→5 是一条完整的红-橙-黄-绿-青绿渐变，3 分不再和正文同色。

<details><summary>核验记录</summary>

逐项复核 clients/web/src/styles/tailwind.css 全部属实：:50-53 亮色 text-primary #2e2845 / secondary #635c78 / muted #6b647d / tertiary #6b647d，muted 与 tertiary 逐字节相同，secondary 与 muted 三通道差 +8/+8/+5；:260-261（及系统暗色副本 :327-328）暗色 muted 与 tertiary 都是 #8f86a5；:66-70 评分色阶里 rating-3 #6b647d 恰等于 :52 text-muted，暗色 :272 rating-3 #a8a0c0 恰等于 :259 text-secondary。混用属实：全仓 text-text-secondary 223 处、text-text-muted 306 处、text-text-tertiary 5 处（全在 modules/course/views/CourseListPage.vue:245/260/323/327/341），所以『删 tertiary 全量替换』的迁移成本极低、可行。对比度我独立算过：#6b647d 相对亮度 0.1372，#dcd8e8 为 0.7016 → 4.02:1（未过 AA），#ece8f4 为 0.8200 → 4.65:1（勉强过线），与发现给的 4.0/4.65 一致；且 bg-bg-tertiary + text-text-muted 的真实组合确实存在于 CommandPalette.vue:36 与 InlineSearch.vue:29 的 text-xs kbd 提示上，不是纸面问题。方案给的 #57506b 我也验了：白底 7.59:1、#dcd8e8 上 5.44:1，与文中 7.5/5.3 吻合。rating-3 改暖黄可行——colors 是通过 design-system/rating.ts 的 ratingColors 供给 RatingBar.vue:32（bar 背景色）、EmojiRating.vue:40（SVG fill）等图形元素，不是正文文字，不引入新的文本对比度问题，且 #d8b84a 在浅底上的对比度（约 1.6:1）与现有 rating-4 #52c07a（约 1.9:1）同量级，不比现状差。

</details>

---

#### `P1` 输入框的聚焦边框和校验错误边框全是隐形的（border-width: 0）

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/SearchPage.vue`
- `clients/web/src/components/common/InlineSearch.vue`
- `clients/web/src/styles/tailwind.css`

**现状**

SearchPage.vue:65 三个输入框的 class 是 `... bg-bg-elevated rounded-lg ... focus:border-primary focus:ring-2 ...`，整串里没有任何 `border` 宽度类；:66 校验失败时追加 `border-danger focus:border-danger focus:ring-danger/20`。InlineSearch.vue:4 容器 class 同样无 border 宽度，:5 却写 `:class="isFocused && 'border-primary shadow-glow-sm'"`。Tailwind v4 preflight 把 `*` 重置为 `border: 0 solid`，而 styles/tailwind.css 的 `@layer base`（:229 起）只定义了主题变量，没有任何 input/select 的基线边框规则。

**问题**

`border-danger` 只设 border-color、宽度仍是 0，所以「至少填一个条件」校验失败时，输入框本身完全没有红色提示，只有下方 SearchPage.vue:174-179 那块独立提示条 —— 用户看不出是哪个字段有问题。同理顶栏搜索框聚焦后只剩一层很淡的 `shadow-glow-sm`（--shadow-glow-sm 是 12px/0.12 透明度），键盘用户几乎看不到焦点落在哪。

**建议改法**

给这几个输入框补 `border border-border-light`（token 已存在）作为基线，让 `focus:border-primary` / `border-danger` 生效；或者干脆复用 tailwind.css:563 已经写好的 `.focus-ring-field` 工具类（它 `border-color` + `box-shadow` 都给了）。校验失败时同时给对应 input 加 `aria-invalid="true"` 和 `aria-describedby` 指向错误文案节点。

<details><summary>核验记录</summary>

逐条核实全部属实。SearchPage.vue:65 的 class 串为 `w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all disabled:opacity-50`，确无任何 border 宽度类；:66 是 `:class="{ 'border-danger focus:border-danger focus:ring-danger/20': validationError }"`；同样的无宽度串还出现在 :91 :111 :144 :163（3 个 input + 2 个 select，「三个输入框」的说法准确）。InlineSearch.vue:4 容器 class 无 border 宽度，:5 为 `:class="isFocused && 'border-primary shadow-glow-sm'"`，:16 输入框本身还写了 border-none outline-none，所以聚焦态唯一反馈就是 --shadow-glow-sm（tailwind.css:94 = `0 0 12px rgba(91,124,247,0.12)`，暗色 :289 为 0.10），确实极弱。tailwind.css:2 是完整 `@import "tailwindcss"`，v4 preflight 对 * 设 `border: 0 solid`；:229 起的 @layer base 通篇只是 [data-theme="dark"] 等变量覆盖，全仓 src/styles 只有 tailwind.css 与仅含一行 @import 的 main.css，无任何 input/select/textarea 基线边框规则（grep 全文只有 :425 的 ::selection 命中）。方案可行性也核过：--color-border-light 确在 @theme :58 定义（暗色 :265/:332），.focus-ring-field 确在 :563-573 且给了 border-color + box-shadow(--shadow-focus-ring :96) + background，现仅被 ReplyForm.vue:5 使用。aria 部分同样缺失：SearchPage 全文无 aria-invalid/aria-describedby，:173-179 的错误块只是普通 div，连 role="alert" 都没有（该页 :29/:222/:305/:347 才有 role="alert"），补 aria 的建议成立。唯一需注记的细节（不影响结论）：SearchPage 的输入框聚焦时 focus:ring-2 仍会渲染一圈 2px、primary@20% 的 box-shadow，所以「完全无焦点提示」只对 InlineSearch 成立；但两处 border 确实恒为 0 宽，校验失败时字段级红框 100% 不可见，问题与方案均成立，P1 合理。

</details>

---

#### `P2` 5 个 UI 原语零引用、10 个动画 token 零引用、tokens.ts 半个文件死代码、两套图标库并存

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/components/ui/Card.vue`
- `clients/web/src/components/ui/Empty.vue`
- `clients/web/src/components/ui/Loading.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/design-system/tokens.ts`
- `clients/web/src/styles/tailwind.css`
- `clients/web/src/modules/home/views/HomePage.vue`

**现状**

①grep 全仓库（排除 components/ui/ 自身和测试）：<Card / <Empty / <Loading / <Pagination / <SearchBar 命中数均为 0，五个组件已在 components.d.ts:19/38/49/57/71 自动注册但没有任何页面用；真正在用的是 components/common/EmptyState.vue（13 处）和 SkeletonCard.vue（9 处）。②tailwind.css:127-144 定义 18 个 --animate-*，排除定义行后实际零使用的有 fade-in-down、scale-in、slide-in-left、slide-in-right、overlay-in、glow-pulse、float、breathe、aurora-drift、border-rotate 共 10 个（后两个的 @keyframes 被 :401/:411/:598 的裸 animation 用了，token 本身没人用），对应 8 个 @keyframes 块完全无人引用。③design-system/tokens.ts 里 colors / animations / glassEffects 三个导出没有任何文件 import（design-system 目录只有 rating.ts 被 7 个组件引用，index.ts 本身零引用）；且 colors 是 8 个硬编码亮色 hex，暗色下全错。④package.json 同时装了 @heroicons/vue 和 lucide-vue-next：lucide 覆盖 52 个 .vue，heroicons 只出现在 Pagination.vue:4 / SearchBar.vue:4 / Empty.vue:2（三个死组件）和 HomePage.vue:11-15（AcademicCapIcon/DocumentTextIcon/UserGroupIcon）。

**问题**

①五个死组件里有三个是暗色坏的（Empty.vue 用 gray-400/600 + 失效的 dark:、Loading.vue 用 gray-200/700 + 本地重复定义了一份已存在于 tailwind.css:195 的 @keyframes shimmer、Pagination/SearchBar 也带 dark:），任何人以为它们能用而直接引入，就会把上面第 2 条的暗色问题再复制一遍；Pagination 还是唯一一份分页实现，而实际有分页需求的 DeveloperAppsPage 是自己手写的。②tokens.ts 号称 single source of truth，但 colors 与 tailwind.css 的 8 个亮色值虽然目前一致，机制上无法同步且不含暗色，是个等着漂移的坑；animations.easing.bounce 在 CSS 里根本不存在。③两套图标库导致首页三个 feature 图标（heroicons，2px 均匀描边）和全站其余 52 个文件（lucide，1.5px 圆头描边）线宽、圆角、视觉重量都不一致，同时多打一个包。

**建议改法**

按 P2 技术债排期，并修正两处执行细节：①删 components/ui/{Card,Empty,Loading,Pagination,SearchBar}.vue 及 components.d.ts 对应条目、删 tokens.ts 的 colors/animations/glassEffects（保留 ratingColors，rating.ts:1 在 import 它）、HomePage.vue:11-15 换 lucide 的 GraduationCap/FileText/Users 后从 package.json 移除 @heroicons/vue —— 这几条照做即可。②清理 --animate-* / @keyframes 时必须与本批第 1 条（Toast 动画）协调：如果采纳 Toast 改用 animate-fade-in-down 工具类的修复，就要保留 --animate-fade-in-down 与 @keyframes fade-in-down，只删 scale-in / slide-in-left / slide-in-right / overlay-in / glow-pulse / float / breathe，否则两条修改互相打架。③话术上不要把这条包装成产物瘦身收益（实测已被摇树），收益是『避免有人引入暗色坏掉的死组件』和『去掉第二套图标库带来的视觉不一致』。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

①删除 components/ui/{Card,Empty,Loading,Pagination,SearchBar}.vue 及 components.d.ts 中对应条目——EmptyState/SkeletonCard 已是事实标准；若日后需要分页，从 DeveloperAppsPage 抽真实需求重写。②删除 tailwind.css 中 8 个无人引用的 @keyframes（fade-in-down、scale-in、slide-in-left、slide-in-right、overlay-in、glow-pulse、float、breathe）及其 --animate-* 声明，aurora-drift/border-rotate 只删 --animate-* token 保留 keyframes。③删除 tokens.ts 里的 colors / animations / glassEffects（glassEffects 的三种效果已由 .glass-card 和 Card.vue 的 scoped 样式覆盖），把 rating.ts 直接从 design-system/index.ts 导出，或干脆去掉 index.ts。④把 HomePage.vue:11-15 换成 lucide 的 GraduationCap / FileText / Users，然后从 package.json 移除 @heroicons/vue。

</details>

<details><summary>核验记录</summary>

四条子事实全部为真：①grep -rE '<(Card|Empty|Loading|Pagination|SearchBar)\b' 与 grep 'ui/Card|ui/Empty|...' 在 src/ 下（排除 components.d.ts）命中 0，真正在用的是 components/common/EmptyState.vue 与 SkeletonCard.vue（各 7 个文件、合计 22 处）；Empty.vue:22-24 用 gray-400/500/600 + dark:，Loading.vue:20-34 用 gray-200/700 且 :44-56 scoped 里重复定义了 tailwind.css:195 已有的 @keyframes shimmer；open-platform/views/DeveloperAppsPage.vue:665-677/1020 确是手写分页。②我逐个 grep 18 个 --animate-*，零使用的正好是 fade-in-down / scale-in / slide-in-left / slide-in-right / overlay-in / glow-pulse / float / aurora-drift / border-rotate / breathe 共 10 个，且 aurora-drift/border-rotate 的 keyframes 确被 tailwind.css:401/411/598 的裸 animation 使用。③design-system/tokens.ts 的 colors/animations/glassEffects 及 index.ts 全仓零 import（唯一被引用的是 rating.ts，被 7 个组件 + modules/review/ratingHelpers.ts 共 8 处引用，发现写『7 个组件』略少但不影响结论）。④package.json:19/26 双装，@heroicons/vue 仅出现在 Pagination.vue:4 / SearchBar.vue:4 / Empty.vue:2 + HomePage.vue:11-15，lucide 覆盖 52 个文件。降级到 P2 的依据：这些全部没有任何用户可见后果或产物体积后果——我用仓库自带 tailwindcss@4.3.3 实际编译了 clients/web/src/styles/tailwind.css，产出的 @keyframes 只有 spin/pulse/fade-in/fade-in-up/vote-bounce/shake/modal-in/shimmer/aurora-drift/border-rotate，8 个无人引用的 keyframes 和对应 --animate-* 变量根本不会进产物（Tailwind v4 按用量摇树）；5 个死组件靠 unplugin-vue-components 按需解析，没被用就不会打包；tokens.ts 的三个导出会被 Rollup 摇掉。所以整条是纯源码卫生/可维护性债，不是 P1 缺陷。另外『Empty.vue 的 dark: 失效』的措辞不准：tailwind.css:5 的 @custom-variant dark 是有效的（实测编译成 :where([data-theme="dark"], ...)），只是在 theme store 的 system 模式下 data-theme 被移除，dark: 才会整体失灵（这属于第 5 条发现的范畴）。

</details>

---

#### `P2` ThemeSwitcher 只做亮/暗二态切换，用户点一次后再也回不到默认的『跟随系统』

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/common/ThemeSwitcher.vue`
- `clients/web/src/stores/theme.ts`

**现状**

stores/theme.ts:9 定义 ThemeMode = 'light' | 'dark' | 'system'，:19-21 默认落到 'system'，:52-59 还专门注册了 matchMedia('(prefers-color-scheme: dark)') 的 change 监听并在 scope dispose 时移除，tailwind.css:296-360 也为系统暗色维护了一整块 60 行的变量副本。但 ThemeSwitcher.vue:29-31 的 toggleTheme 是 themeStore.setMode(themeStore.isDark ? 'light' : 'dark')——只在两个显式值之间跳，永远不会写回 'system'；:23-27 的 nextLabel 也只有 switchToLight / switchToDark 两条文案。全站 grep 没有第二个调用 setMode 的地方。

**问题**

'system' 是默认值也是唯一会跟随 OS 自动切换的模式，但它是**单向的**：用户第一次点击图标（哪怕只是想看看效果）之后 localStorage 就被写死成 light 或 dark，此后 OS 白天/夜间自动切换对这个站完全失效，且 UI 上没有任何入口能回到 system——只能手动清 localStorage。结果是 theme.ts 里那套 mql 监听、tailwind.css 里那 60 行 prefers-color-scheme 副本（还因为漏了 --shadow-focus-ring 而与 [data-theme=dark] 不同步）只对『从没点过主题按钮的用户』生效，投入产出严重不匹配。另外按钮当前只有 aria-label，读屏用户听不出当前处于哪个模式。

**建议改法**

改成三态循环：toggleTheme 走 light → dark → system → light，图标对应 Sun / Moon / Monitor（lucide 有 Monitor），nextLabel 补第三条 i18n 文案（zh-CN/en-US 都要加），并给按钮加 :aria-label 说明当前模式而不只是下一个模式。如果产品上不打算支持三态，那就反过来彻底删掉 'system'：store 只留 light|dark（首次进入按 matchMedia 初始化一次即可），删掉 mql 监听、tailwind.css:296-360 整块 media 副本和 :404-413 的 aurora 副本、以及 :8 那个没人用的 dark-system 变体——能一次性去掉约 80 行永远和主块对不齐的重复维护。

<details><summary>核验记录</summary>

逐条核对属实：clients/web/src/stores/theme.ts:10 定义 ThemeMode = 'light' | 'dark' | 'system'（发现写 :9，差 1 行可接受），:18-22 无有效存储时默认 'system'，:52-59 注册 matchMedia change 监听并在 safeOnScopeDispose 里移除；clients/web/src/components/common/ThemeSwitcher.vue:29-31 toggleTheme 就是 setMode(isDark ? 'light' : 'dark')，:23-27 nextLabel 只有 switchToLight/switchToDark 两条，且全仓 grep setMode 只有 ThemeSwitcher.vue:30 这一个调用点，确认 'system' 一旦离开就没有任何 UI 能回去（setMode 会写 localStorage['theme-mode']，theme.ts:34-37）。重复维护也属实：tailwind.css:296-360 是 [data-theme="dark"]（:230-293）的整块副本，且副本里确实漏了 --shadow-focus-ring（主块 :291 有，副本 :349-357 只到 shadow-glow-accent），:404-413 是 aurora 副本，:8 的 dark-system 变体全仓零使用（grep 只命中定义行本身）。方案可行性也验过：lucide-vue-next 有 Monitor 图标（node_modules/lucide-vue-next/dist/esm/icons/monitor.js 存在），i18n 键位于 src/i18n/locales/zh-CN/common.ts:36-39 与 en-US/common.ts:37-38，加第三条文案是小改动。补充一点支持该发现的证据：applyTheme 在 system 模式下 removeAttribute('data-theme')（theme.ts:41-42），而 dark 变体实测编译为 :where([data-theme="dark"], ...)，classList.toggle('dark') 并不匹配它——所以 system+OS 暗色的用户，全站 31 处 dark: 工具类都不会生效，只有 CSS 变量翻转，进一步说明这块双轨维护确实收益为负。P2 定级与描述均无需调整。

</details>

---

### 导航与信息架构

共 14 条：P0 0 / P1 5 / P2 9

#### `P1` FloatingModuleNav 与顶部主导航重复，移动端退化成一个语义不明、无法关闭的圆钮

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/components/layout/FloatingModuleNav.vue:103-107`
- `clients/web/src/components/layout/FloatingModuleNav.vue:125-127`
- `clients/web/src/components/layout/FloatingModuleNav.vue:190-197`
- `clients/web/src/components/layout/AppShell.vue:19`
- `clients/web/src/components/layout/AppHeader.vue:261-266`

**现状**

AppShell.vue:19 在每一页都挂 FloatingModuleNav。它的 tabDefs（:103-107）是 评课 /courses/reviews、教师 /teachers、资源 /resources；顶部主导航（AppHeader.vue:261-266）是 首页 /、课程 /courses、教师 /teachers、资源 /resources —— /teachers 和 /resources 两项完全重复。MOBILE_BREAKPOINT = 768（:125），`showExpandedTabs = expanded && !isMobileViewport`（:197），所以 <768px 时永远只渲染 1 个 40px 圆钮；两处 tooltip 都是 `opacity-0 group-hover:opacity-100`（:34、:71），触屏根本不触发。activeTab 兜底 `|| tabs.value[0]`（:191），首页/资源页以外的路径匹配不上时显示「评课」图标并套 `text-primary`。

**问题**

桌面端：同一屏出现两套指向相同路由的导航，用户不知道该用哪个，而且它可拖拽、位置写进 localStorage（floating-nav-position），拖歪了没有「恢复默认」。移动端：右下角固定一个没有文字、tooltip 永不出现的圆钮，点它会跳到「当前模块」——在首页上那是「评课」，等于一个会把人随机带走的按钮；它还常驻遮挡右下角内容区，没有任何收起/关闭入口。

**建议改法**

首选直接删除：移除 clients/web/src/components/layout/FloatingModuleNav.vue（288 行）和 AppShell.vue:19 的挂载，导航责任全部收回顶部 header（配合下面第 3、7 条把 header 补全）。如果确实要保留快捷入口，改成单一「发测评」FAB：只在 `route.path.startsWith('/courses') || route.path.startsWith('/teachers')` 显示，固定右下 + `env(safe-area-inset-*)`，不可拖拽、不存 localStorage，带可见文字标签（不是纯图标），点击走 ensureCanPostReview。

<details><summary>核验记录</summary>

逐行核实全部属实：AppShell.vue:19 全局挂载；tabDefs :103-107 = 评课/教师/资源，与 AppHeader.vue:261-266 的 教师//resources 两项重复；MOBILE_BREAKPOINT=768 (:125)、showExpandedTabs = expanded && !isMobileViewport (:197) 使 <768px 只剩 1 个 40px 圆钮；两处 tooltip 均为 opacity-0 group-hover:opacity-100 (:34、:71)，触屏不触发；activeTab 兜底 `|| tabs.value[0]` (:190-192) 且图标恒为 text-primary (:28)，故首页显示「评课」并跳走；localStorage 键 floating-nav-position (:123) 无恢复默认；移动端无任何收起入口。补充两点供实施参考：(a) 移动端已经不可拖拽（startDrag :208 提前 return）且已用 env(safe-area-inset-*) 定位（:172-183），方案中这两条属于「已实现」；(b) 若直接删除组件，/courses/reviews 将只剩 HomePage.vue:95 的 feature 卡片一个入口，必须与第 3 条同批落地。

</details>

---

#### `P1` 头部用户菜单用整页跳转打开同源 SPA 路由，每次点都白屏重载

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/layout/AppUserMenu.vue`
- `clients/web/src/utils/redirect.ts`

**现状**

AppUserMenu.vue:288-294 的 goToIdentityRoute() 对 6 个菜单项（账号资料/账号安全/开发者应用/身份认证/学生认证/QQ 绑定，见 AppUserMenu.vue:176-183 的 identityMenuRoutes）执行 navigateToExternalURL(accountCenterURL(path))；而同一菜单里的 goToUser()（AppUserMenu.vue:284-286）用的是 router.push('/user/reviews')。accountCenterURL() 走 utils/redirect.ts:148-150 → configuredWebOrigin()，后者在 utils/redirect.ts:25-29 未配置 VITE_WEB_URL 时直接回落到 window.location.origin。

**问题**

单站部署（默认）下这些路径就是本 SPA 已注册的路由（router/index.ts:147-168、:430-472），却被当成外链做 window.location 整页导航：白屏重载、丢失滚动位置与内存态、重跑鉴权引导、重连通知 SSE，还会让浏览器返回键行为和 SPA 内跳转不一致。同一个菜单里 7 项有 1 项是软跳、6 项是硬跳，用户体感随机。

**建议改法**

goToIdentityRoute 先判断目标 origin：accountCenterURL(path) 的 origin === window.location.origin 时走 router.push(path)，仅在真正跨 origin（部署了独立账号中心）时才 navigateToExternalURL。同一判断可复用到 useNotificationBellController.ts:278-283 的通知跳转分支，保证「点通知」和「点菜单」的跳转语义一致。

<details><summary>核验记录</summary>

核实属实：AppUserMenu.vue:288-294 goToIdentityRoute 对 :175-182 的 6 条路径一律 navigateToExternalURL(accountCenterURL(path))，而同菜单 :283-286 goToUser 用 router.push('/user/reviews')；utils/redirect.ts:196-198 即 window.location.assign。机制描述略有偏差但结论不变：configuredWebOrigin() 在未配 VITE_WEB_URL 且非 join 域时返回 null（:25-29 + :31-45），回落发生在 absoluteURLOnPreferredOrigin(:130) 里，最终仍是同源绝对 URL → 整页重载；生产配置 WEB_VITE_WEB_URL=https://stuhelper.com 与主站同源（docs/guides/production-topology.md），所以线上默认也是硬跳。这些路径都在本 SPA 注册（router:147-167、:430-472），重载会丢滚动位置、重跑 bootstrap 与通知 SSE 重连。origin 比较方案安全：join.stuhelper.com 下 deriveWebOriginFromCurrentLocation 会解析出跨域主站，仍走硬跳，符合 router/join-domain.ts:24-29 的拦截设计。实施提醒：components/layout/__tests__/AppUserMenu.test.ts 中三条 'opens ... on the account center from the main host' 用例（断言 navigateToExternalURL 被调用且 routerPush 未被调用）需要同步改成按 origin 分支断言。

</details>

---

#### `P1` 枢纽页主 CTA「写测评」绕过能力检查，路由守卫把用户静默弹回首页

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/course/views/TeachingHubPage.vue:161-170`
- `clients/web/src/router/index.ts:620-628`
- `clients/web/src/components/layout/AppHeader.vue:76-89`
- `clients/web/src/composables/useReviewPost.ts:175-190`

**现状**

AppHeader 的「写测评」按钮走 handleWriteReview() → ensureCanPostReview()，缺能力时会 toast.error 并把用户送到实名/学生认证页（useReviewPost.ts:180-188）。但同一屏上 TeachingHubPage:161-170 的第二张 CTA 卡片是裸 `<router-link :to="{ name: 'course-review-post' }">`，直接进路由。守卫在 router/index.ts:620-628 判定 requiredCapabilities 不足后 `return { name: "home" }`。

**问题**

未获得 REVIEW_CREATE 的登录用户（未实名/未学生认证/被限制）点 hub 页最显眼的强调色 CTA，页面闪一下就回到首页，没有 toast、没有目标页、没有原因。用户唯一能做的是再点一次，再被弹一次。同一页两个「写测评」入口行为完全不同，更难自查。直接输入 URL、书签、外部分享链接进入 /courses/reviews/post 也是同样的静默失败。

**建议改法**

1) TeachingHubPage:161-170 改成 `<button @click="handleWriteReview">`，复用 useReviewPost 的 ensureCanPostReview（可把 AppHeader.vue:284-293 的 handleWriteReview 抽到 useReviewPost 里共享）。2) 守卫不要静默回首页：新增 `/forbidden` 路由（modules/errors 下已有 NotFoundPage 可对照），返回 `{ name: 'forbidden', query: { from: to.fullPath } }`，页面上写清缺哪个能力 + 一个「去认证」按钮；最低限度也要 `{ name: 'home', query: { denied: String(to.name) } }` 并在 HomePage 读取 query 弹 toast。

<details><summary>核验记录</summary>

事实全部核实无误：TeachingHubPage.vue:161-170 确实是裸 <router-link :to="{ name: 'course-review-post' }">；router/index.ts:620-628 在 requiredCapabilities 不足时 `return { name: "home" }`，全局无 afterEach/toast 兜底，属于真静默；AppHeader.vue:76-89 + :284-293 走 ensureCanPostReview，useReviewPost.ts:176-191 会 toast.error 并跳实名/学生认证页。两个入口行为确实不一致，且 /courses 上两者同屏（showWriteReview = path.startsWith('/courses')）。方案可行（modules/errors 下确有 NotFoundPage 可对照）。仅严重度需下调：未登录用户走的是 :611-613 的 login+redirect 分支（正常），受影响的只是「已登录但缺 REVIEW_CREATE」的用户，且同屏还有一个能正确引导到认证页的 header 按钮，属于「重复入口坏了一个」而非核心功能完全阻塞，不构成 P0。

</details>

---

#### `P1` 顶栏搜索按 Enter 是死路：没有「查看全部结果」，超出 top-10 的课搜不到

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/common/InlineSearch.vue`

**现状**

InlineSearch.vue:161 固定请求 `searchCourses(trimmed, 10, ...)`，下拉最多 10 条。:216-222 的 Enter 分支只在 `activeIndex >= 0` 时 `selectCourse` / `selectRecent`；`activeIndex` 初值 -1（:126）且每次输入变化都被重置回 -1（:140）。下拉里没有任何「查看全部 / 高级搜索」条目。

**问题**

用户输入「数学」按回车 —— 什么都不发生，页面毫无反馈，看起来像坏了。而「数学」这种宽泛词命中远超 10 条，第 11 条之后的课程在这个入口里永远够不着，用户也没有任何提示告诉他还有更多结果、去哪儿看。

**建议改法**

1) Enter 且 `activeIndex === -1` 时 `router.push({ path: '/search', query: { courseName: query.trim() } })`；2) 在下拉结果末尾固定一条「查看全部 N 个结果」item（纳入方向键 itemCount 计算），点击同样跳 `/search`；3) 结果条数不足时把后端返回的 total 显示出来，让用户知道被截断了。

<details><summary>核验记录</summary>

逐行核实无误：InlineSearch.vue:161 `api.course.searchCourses(trimmed, 10, { signal })`（shared/src/api/courses.ts:27-42 将数字参数归一化为 `pageSize: 10`）；:207-227 handleKeydown 的 Enter 分支在 :216-222，两条件都要求 `activeIndex.value >= 0`，:126 初值 -1、:139-140 每次 query 变化都重置回 -1，且 :217 无条件 preventDefault，所以未用方向键选中时按 Enter 确实纯无反应；:33-92 下拉面板只有结果项与最近搜索项，无「查看全部/高级搜索」条目。方案可行且与既有实现一致：/search 路由存在（router/index.ts:361-362），SearchPage.vue:545 的 hydrate 支持 courseName/q；TeachingHubPage.vue:317-331 已实现完全相同的「Enter 无选中项则 push 到 advancedSearchRoute」模式，此改动属于把 hub 的行为对齐到顶栏；第 3 点也可行 —— 后端返回体含 total（shared/src/api/courses.ts 走 /courses/search，SearchPage 用 readCoursePagePayload 取 total），只是 InlineSearch 现在用 readCourseListPayload 把 total 丢掉了，改用带 total 的解析即可。P1 合理。

</details>

---

#### `P1` 首页对登录用户等于「满屏 Hero + 全站导航的第三份副本」，且 footer 只有首页有

> 核验确认　|　工作量：L

**位置**

- `clients/web/src/modules/home/views/HomePage.vue`
- `clients/web/src/components/business/HeroSection.vue`
- `clients/web/src/components/layout/FloatingModuleNav.vue`
- `clients/web/src/components/layout/AppHeader.vue`

**现状**

HomePage.vue 全文 193 行没有任何登录态分支。首屏是 HeroSection，HeroSection.vue:52 写死 min-height: 100vh。往下 :143-151 的三张 FeatureCard 指向 /courses/reviews、/teachers、/resources，而 FloatingModuleNav.vue:104-106 的三个 tab 恰好是同样这三个地址，AppHeader.vue:262-265 的 navItems 也是 /、/courses、/teachers、/resources。再往下是统计数字，最后 :182-191 是一个内联 footer，链到 /about、/privacy、/terms——全仓库没有任何 Footer 组件，components/layout 下也搜不到 footer。

**问题**

① 一个已登录的老用户打开首页，要滚过整整一屏纯装饰的 Hero，才能看到三张卡片，而这三张卡片去的地方和常驻在屏幕上的浮动导航、顶部导航完全一样——首页没有提供任何顶栏做不到的事，纯属一次多余的点击；② 它也不认人：不知道你收藏过哪些课、写过哪些评价、关注的老师有没有新评价，对一个校园评课平台来说，首页本该是「最新评课 / 热门课程 / 我的收藏」的落点；③ footer 只存在于首页，意味着用户在 /courses、/teachers 等任何页面都找不到隐私政策和服务条款的入口——对有 UGC 的平台这是不该有的缺口。

**建议改法**

① 登录态下把 HeroSection 换成一条紧凑的欢迎条（一行标题 + 搜索框），把首屏让给内容；② 用已有接口填三块真内容——api.review.getHotCourses（TeachingHubPage.vue:523 已在用）做「热门课程」、评价流做「最新评课」、收藏列表做「我的收藏」入口，删掉与导航重复的三张 FeatureCard；③ 游客态保留现有 Hero + 卡片作为落地页；④ 把 HomePage.vue:182-191 的 footer 抽成 components/layout/AppFooter.vue，挂到 AppShell 上，让 /privacy 和 /terms 全站可达。

<details><summary>核验记录</summary>

核实无误：HomePage.vue 全文 193 行，无 useAuthStore、无任何登录态分支；HeroSection.vue:52 min-height:100vh 写死；HomePage.vue:89-116/143-151 三张 FeatureCard 指向 /courses/reviews、/teachers、/resources，与 FloatingModuleNav.vue:104-106 三个 tab 完全同址，AppHeader.vue:261-265 navItems 为 /、/courses、/teachers、/resources。footer 仅在 HomePage.vue:182-191；全仓库 grep 无 AppFooter/Footer 组件，AppShell.vue 只有 AppHeader + main + CommandPalette + FloatingModuleNav，且 /privacy、/terms（router:234-251）除首页 footer 外全站零入口——这条对 UGC 平台是实打实的缺口。方案②引用的 api.review.getHotCourses 确在 TeachingHubPage.vue:523 使用。P1 合理。

</details>

---

#### `P2` /courses 三层入口把课程列表埋掉：全站只有一个链接指向 /courses/list

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/web/src/router/index.ts:255-269`
- `clients/web/src/modules/course/views/TeachingHubPage.vue:136-145`
- `clients/web/src/components/layout/AppHeader.vue:263`

**现状**

顶部导航「课程」指向 /courses = TeachingHubPage（枢纽页，router/index.ts:255-261）。真正的课程列表在 /courses/list（:263-269），全仓库只有 TeachingHubPage.vue:137 这一处 `to="/courses/list"` 引用它。hub 页上方是两张 CTA 卡片（浏览课程 / 写测评，:123-173），下方「热门课程」网格（:180-198）直接跳 /courses/:id 详情。

**问题**

用户想看课程列表：点顶部「课程」→ 落到 hub → 向下滚过 hero 和统计 → 找到「浏览课程」卡片 → 点「查看全部」。2 次点击 + 一次滚动，且第二次点击的按钮文案是「查看全部」而不是「课程列表」。hub 页的热门课程又把人直接送去详情，列表页事实上被架空成一个孤儿页。而 /courses/reviews（评课流）在顶部导航里完全不可见，只能靠那个浮动圆钮到达。

**建议改法**

压平成两层：把 /courses 直接指向 CourseListPage，hub 的两张 CTA 卡片降级为列表页顶部一行工具条（「写测评」主按钮 + 「评课流」次级链接），热门课程作为列表页的置顶分组。若要保留 hub，则把顶部导航「课程」改成带二级 Tab 的页面：进入 /courses 后 header 下方常驻 `课程列表 | 评课流 | 关于` 三个平级 Tab（复用 components/common/TabBar.vue），三个入口一次可见，点击各自切换 /courses/list、/courses/reviews、/courses/about。

<details><summary>核验记录</summary>

路由与链接事实属实：/courses→TeachingHubPage (router:255-261)、/courses/list (:263-269) 全仓仅 TeachingHubPage.vue:137 一处引用（grep 确认，另一处是测试）；CTA 卡片 :123-173、热门课程 :180-198 直跳详情；components/common/TabBar.vue 存在，方案可行。但有两处需要修正：(1)「/courses/reviews 只能靠那个浮动圆钮到达」不成立——HomePage.vue:95 的 feature 卡片同样链到 /courses/reviews；(2)「向下滚过 hero 和统计」中 hub 页并无统计网格（统计在 HomePage），hub 首屏是 hero+大搜索框，CTA 卡片位置比描述靠前。剔除夸大后，这是「列表页入口偏深 + 评课流无顶部导航位」的 IA 优化，不是功能缺陷，降为 P2。

</details>

---

#### `P2` AppHeader 移动菜单缺点击外部关闭 / Esc / 焦点管理 / 滚动锁，同一 header 里的 AppUserMenu 却全都有

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/layout/AppHeader.vue:94-105`
- `clients/web/src/components/layout/AppHeader.vue:124-180`
- `clients/web/src/components/layout/AppHeader.vue:304-322`
- `clients/web/src/components/layout/AppUserMenu.vue:255`
- `clients/web/src/components/layout/AppUserMenu.vue:319-332`

**现状**

mobileMenuOpen 只有三个关闭路径：再点一次汉堡按钮（:101）、点菜单内的链接（:147 closeMobileMenu）、以及 `watch(() => route.fullPath)`（:317-322）。没有 document click 监听、没有 keydown Escape、没有焦点陷阱、没有 body 滚动锁。对照 AppUserMenu.vue:319-332 注册了 `document.addEventListener('click', onClickOutside, true)`，:255/:275 处理了 Escape。仓库里已存在 composables/useBodyScrollLock.ts 和 useDialogFocus.ts，AppHeader 一个都没用。

**问题**

同一个 header 上两个下拉行为完全相反：点头像下拉后点空白处会关，点汉堡菜单后点空白处不会关——移动端用户最直觉的关闭操作在这里失效，只能精准点回那个 44px 的汉堡按钮。菜单打开时按 Tab 会直接跳进菜单后面的页面内容（菜单是 absolute 覆盖层但不拦焦点），键盘用户会「消失」在看不见的元素上；背景仍可滚动，菜单浮在半空。

**建议改法**

只补两件事：(1) onMounted 注册 document click（比较 event.target 是否在 #app-mobile-nav 与汉堡按钮内）关闭 mobileMenuOpen；(2) 面板容器上 @keydown.esc 关闭并 nextTick 把焦点还给汉堡按钮（可与 AppUserMenu 共抽 usePopoverDismiss）。不要引入 useDialogFocus/useBodyScrollLock——该菜单是非模态 disclosure，焦点陷阱与滚动锁不适用；也不要删除 :166-178 的登录项，<640px 时 header 登录按钮只剩图标（max-sm:hidden 掉了文字），菜单里那条是唯一带文案的登录入口。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

直接复用 AppUserMenu 的模式：在 AppHeader 里加 `onMounted(() => document.addEventListener('click', onClickOutside, true))`（比较 `event.target` 是否在 `#app-mobile-nav` 和汉堡按钮内）+ `@keydown.esc` 关闭并把焦点还给汉堡按钮；给 :124 的容器接 useDialogFocus 做焦点循环、接 useBodyScrollLock 在 mobileMenuOpen 为 true 时锁滚动。顺带删掉 :166-178 那条 `sm:hidden` 的登录项——<640px 时 header 里的登录按钮（:114 `max-sm:w-11`）本来就还在，两个登录入口同时出现。

</details>

<details><summary>核验记录</summary>

点击外部与 Esc 的缺失属实：AppHeader.vue 全文无 document click 监听、无 keydown，关闭路径只有汉堡按钮 :101、菜单内链接 :147、route.fullPath watch :317-322；对照 AppUserMenu.vue:314-320/:331-333 有 capture 阶段 onClickOutside，:255-258 与 :275-277 有 Escape。这两点应修。但其余论据不成立或过度处方：(1) 该菜单是非模态 disclosure 下拉（absolute 面板、无遮罩、背景可见可交互），WAI-ARIA 不要求焦点陷阱与 body 滚动锁，Tab 走到后面的页面内容是该模式的正常行为，把 useDialogFocus/useBodyScrollLock 套上去反而与 ModerationDialog 等真模态混淆语义；(2)「只能精准点回那个 44px 的汉堡按钮」低估了现状——:103 在打开时图标已切换为 X，是可见的关闭态；(3) 建议删掉 :166-178 的 sm:hidden 登录项是倒退：<640px 时 header 内的登录按钮因 max-sm:hidden(:119) 只剩图标无文字，删掉菜单里那条带文案的入口会让登录入口失去可读标签。方案应收敛为「加 click-outside + Esc 关闭并把焦点还给汉堡按钮」。

</details>

---

#### `P2` 两个「关于」页并存，首页 hero 和 footer 各指一个；/courses/about 只有一个入口还占着 /courses 命名空间

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/home/views/HomePage.vue:128-130`
- `clients/web/src/modules/home/views/HomePage.vue:186`
- `clients/web/src/router/index.ts:226-233`
- `clients/web/src/router/index.ts:271-277`

**现状**

HomePage.vue:128-130 的 onLearnMore（hero 区「了解更多」）push 到 /courses/about → modules/review/views/AboutPage.vue；同一页 :186 的 footer「关于」链到 /about → modules/common/views/InfoPage.vue（pageKey='about'，router:226-233）。TeachingHubPage.vue:109 也链 /about。/courses/about 在全仓库只有 HomePage:129 一个入口。

**问题**

同一个首页上两个「关于」落到两份不同内容，用户无法预期点哪个会看到什么，也不知道哪份是权威说明。/courses/about 还挤在 /courses 命名空间里，是第 4 条那个顺序陷阱的直接受害候选（它靠排在 271 行才活着）。同时它是一个只有单入口的孤儿页，维护成本对不上访问量。

**建议改法**

合并到 /about：把 review/views/AboutPage.vue 里关于评课规则的内容并进 InfoPage 的 about 文案，删除 router/index.ts:271-277 的 course-about 路由和 modules/review/views/AboutPage.vue，HomePage.vue:129 改成 `router.push('/about')`。若评课规则确实需要独立页，改挂到 /courses/reviews/about，并在评课流页面顶部给出显式入口，不要只从首页 hero 到达。

<details><summary>核验记录</summary>

全部核实属实：HomePage.vue:128-130 onLearnMore → router.push('/courses/about')（→ modules/review/views/AboutPage.vue，372 行，含前言/FAQ），同页 footer :186 的「关于」→ /about（→ modules/common/views/InfoPage.vue，pageKey='about'，router:226-233）；/courses/about 注册于 router:271-277；全仓 grep 'courses/about' 只有 HomePage.vue:129 一个入口。同一首页两个「关于」落到两份不同内容属实，P2 合适。注意实施成本：AboutPage 是 372 行带 FAQ 分节的模板，合并进 89 行的 InfoPage 需要同时迁移 review.about.* 文案结构，备选方案（挪到 /courses/reviews/about 并在评课流页给显式入口）成本更低，可优先。

</details>

---

#### `P2` 全局搜索没有任何可见入口：CommandPalette 只能 Cmd+K，InlineSearch 只在 /courses 出现

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/components/layout/AppShell.vue:18`
- `clients/web/src/composables/useCommandPalette.ts:38-42`
- `clients/web/src/composables/useCommandPalette.ts:66-77`
- `clients/web/src/components/layout/AppHeader.vue:69-74`
- `clients/web/src/components/layout/AppHeader.vue:236`

**现状**

CommandPalette 全局挂载在 AppShell.vue:18，但 useCommandPalette 导出的 open()/toggle() 在整个 clients/web/src 里没有任何 UI 调用点——唯一触发是 useCommandPalette.ts:38 的 `(metaKey||ctrlKey) && k`。header 里的 InlineSearch 条件是 `v-if="showCourseSearch"`，而 showCourseSearch = `route.path === '/courses'`（:236），连课程列表页 /courses/list 都没有。CourseListPage 另有一套自己的本地拼音过滤（usePinyinSearch），/search 又是第四套。

**问题**

触屏 / 移动端用户永远打不开命令面板（没有按键可按）；桌面用户没有任何视觉提示知道存在 Cmd+K，UI 上零处出现「⌘K」字样。结果是全站有 4 套搜索实现，但在首页、教师页、资源页、课程列表页、课程详情页——也就是绝大多数页面——用户一个搜索框都看不到。

**建议改法**

只给 CommandPalette 补一个可见触发点：header 右侧常驻按钮（<lg 放大镜图标、≥lg 假输入框样式 + <kbd>⌘K</kbd>），点击调 useCommandPalette().toggle()。不要放宽 showCourseSearch —— /courses/list、/teachers、/resources 已各自有页内搜索框（CourseListPage.vue:209、ResourceListPage.vue:269 等），再加 header 搜索会造成同屏双搜索框。InlineSearch 与 CommandPalette 的收敛可作为独立的后续项评估。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

header 常驻一个搜索入口：<lg 显示放大镜图标按钮，≥lg 显示一个假输入框样式的按钮（占位文案 nav.searchCoursePlaceholder + 右侧 `<kbd>⌘K</kbd>`），点击调用 useCommandPalette().toggle()。同时把 showCourseSearch 放宽为 `route.path.startsWith('/courses') || route.path.startsWith('/teachers')`。收敛实现：InlineSearch(296 行) 与 CommandPalette(345 行) 功能重叠，保留 CommandPalette 作为全局入口，把 InlineSearch 降级为纯页面内筛选或直接删除。

</details>

<details><summary>核验记录</summary>

前半段属实且值得修：AppShell.vue:18 全局挂载 CommandPalette，但全仓 grep 显示 useCommandPalette 的 open()/toggle() 没有任何 UI 调用点（CommandPalette.vue:123 只解构了 close），唯一触发是 useCommandPalette.ts:38 的 metaKey/ctrlKey+k，触屏用户确实永远打不开，UI 上也无 ⌘K 提示。但「绝大多数页面用户一个搜索框都看不到」是错的：CourseListPage.vue:209-231 有可见搜索输入框（usePinyinSearch + URL 同步）、TeacherHubPage 有搜索框（:111-121 handleSearchInput）、ResourceListPage.vue:269-303 有搜索+标签输入、/courses 的 TeachingHubPage.vue:18-58 首屏就是大搜索框。真正没有搜索入口的只有首页和详情页。据此：问题应重述为「CommandPalette 是一个没有任何 UI 可达性的死功能」，P2；并且不应放宽 showCourseSearch 到 /teachers、/courses/*，那会与这些页面已有的页内搜索重复两个搜索框。

</details>

---

#### `P2` 移动菜单顶部偏移写死 56px：小屏浮空 12px，820–1023px 凭空断开一条 56px 空缝

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/layout/AppHeader.vue:128-132`
- `clients/web/src/components/layout/AppHeader.vue:182`
- `clients/web/src/styles/tailwind.css:118-124`

**现状**

菜单容器的 top 用内联 style 算：`showCourseSearch ? 'calc(var(--navbar-height) + 56px)' : 'calc(var(--navbar-height) + 8px)'`（:128-132）。而 showCourseSearch 只是 `route.path === '/courses'`（:236），第二行搜索框是 `class="hidden max-tablet:block"`（:182），只在 <820px 渲染。token 里 --navbar-height: 56px、--mobile-header-extra-height: 44px、--mobile-header-height = 100px（tailwind.css:118-120），--breakpoint-tablet: 820px（:124）。

**问题**

两个具体故障：(a) <820px 在 /courses 上，header 实际高 100px，菜单被推到 112px，中间浮空 12px；(b) 820–1023px（汉堡按钮 max-lg 仍显示，搜索行已被 `hidden` 关掉）在 /courses 上，菜单仍下移 56px，菜单和 header 之间断开一条 56px 的透明缝，看起来像渲染错误。56 这个数字在 token 里根本不存在，是第三个互相冲突的高度常量。

**建议改法**

偏移改用已有 token 并让判定跟着视口走：把内联 style 换成 class —— 默认 `top-[calc(var(--navbar-height)+8px)]`，在 /courses 上加 `max-tablet:top-[calc(var(--mobile-header-height)+8px)]`。更稳的做法是把菜单容器从 `absolute + top` 改成挂在 header 内容流末尾用 `top-full mt-2`，由浏览器按 header 实际高度定位，彻底删掉这段 JS 计算。

<details><summary>核验记录</summary>

数值链路核实无误：AppHeader.vue:128-132 内联 top 计算、:236 showCourseSearch = (route.path === '/courses')、:182 第二行搜索 hidden max-tablet:block；tailwind.css:117-124 中 --navbar-height:56px、--mobile-header-extra-height:44px、--mobile-header-height=100px、--breakpoint-tablet:820px；构建产物 dist/assets/index-*.css 中 max-tablet 编译为 @media not all and (min-width:820px)，确认 820–1023px 区间搜索行不渲染而 56px 偏移仍生效（汉堡按钮 max-lg 到 1023px 仍显示），此时菜单与 header 之间确实断开 56px。<820px 一路径下 header 实高 100px、菜单 top 112px，实际偏差是 12px 对设计意图 8px，即多 4px。综合看这是单一路由、单一视口带内的纯视觉错位，无功能与可达性影响，P1 偏高，降 P2。方案本身可行（改用 max-tablet:top-[calc(var(--mobile-header-height)+8px)] 或改 top-full mt-2）。

</details>

---

#### `P2` 课程/教师深层页面没有任何可见的返回上一级路径，而 /user/* 全都有——同一 app 两套约定

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/review/views/CourseDetailPage.vue:1-40`
- `clients/web/src/modules/course/views/CourseListPage.vue`
- `clients/web/src/modules/user/views/AccountProfilePage.vue:20`
- `clients/web/src/modules/resource/views/ResourceDetailPage.vue:203`
- `clients/web/src/router/index.ts:331-345`

**现状**

grep 全仓库 breadcrumb 为 0 处。/user/* 系列（AccountProfilePage.vue:20、AccountSecurityPage.vue:24、AuthorizedAppsPage.vue:20、IdentityVerificationPage.vue:11、PhoneBindingPage.vue:10、QQBindingPanel.vue:10）、资源详情（ResourceDetailPage.vue:203）、搜索页（SearchPage.vue:16）都有 ArrowLeft 返回。CourseDetailPage.vue 模板从第 1 行的 CourseThemeProvider 直接进 loading/error/course header（:1-40），没有返回；CourseListPage.vue、TeacherProfilePage.vue 同样为 0。

**问题**

首页热门课程卡片、hub 页热门课程网格都直接跳 /courses/:id，用户落在详情页后，页面内没有任何回到课程列表或 /courses 的可点元素，只能按浏览器后退。从分享链接/书签直接打开详情页时，后退等于离站。同一产品里「用户中心有返回箭头、课程详情没有」也是明显的不一致。

**建议改法**

抽一个 PageBackLink 组件放 components/common/：接受 `:to` 和上级标题（用 `:to` 而不是 router.back()，深链接进入时才有正确去向），沿用 AccountProfilePage.vue:20 已有的视觉样式。上级关系写进路由 meta，例如 course-detail: `meta: { parent: 'course-list' }`、teacher-profile: `meta: { parent: 'teacher-hub' }`，组件读 meta.parent 解析标题（可复用 routes.* 的 titleKey，和 usePageMeta 共用同一份文案）。先覆盖 course-detail / course-reviews / teacher-profile / course-list 四个页面。

<details><summary>核验记录</summary>

事实基本属实：全仓 grep breadcrumb 为 0；AccountProfilePage.vue:16-22、AccountSecurityPage.vue:24、AuthorizedAppsPage.vue:16-22、IdentityVerificationPage.vue:4-12、PhoneBindingPage.vue:10、QQBindingPanel.vue:10、AcademicInfoPage.vue:3-11、ResourceDetailPage.vue:203、SearchPage.vue:16 都有 ArrowLeft 返回；CourseDetailPage.vue 模板 :1-45 从 CourseThemeProvider 直接进 loading/error/course header，全文无 ArrowLeft；CourseListPage.vue 无；TeacherProfilePage.vue 只有 :226 的 notFound 分支返回首页。但「只能按浏览器后退」不准确：AppShell 的固定 header 在每一页常驻，桌面端有「课程」主导航项（AppHeader.vue:263），移动端有汉堡菜单，深链接进入也能上行。这是一致性/IA 缺口而非死路，P1 偏高。方案（PageBackLink + meta.parent）可行。

</details>

---

#### `P2` 通知铃铛面板无 Escape 关闭、无焦点管理，和同一头部里的用户菜单是两套无障碍标准

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/common/NotificationBell.vue`
- `clients/web/src/components/layout/AppUserMenu.vue`

**现状**

NotificationBell.vue:88-96 只注册了 document click 关闭（handleDocumentClick 见 useNotificationBellController.ts:289-298），没有任何 keydown 处理：按 Esc 关不掉面板，打开后焦点仍留在铃铛按钮上、Tab 才会走进面板，关闭后也没有焦点归位；面板容器是 role="region"（NotificationBell.vue:26），条目是普通 button（NotificationItem.vue:159-163），没有 menu/listbox 语义；路由跳转后 showPanel 也不会被重置（useNotificationBellController.ts:273 只在点击条目时置 false）。同一个 header 里的 AppUserMenu.vue:255-258、:275-277 却完整实现了 Escape 关闭 + 焦点归还 + role="menu"/menuitem + 上下键 roving（AppUserMenu.vue:40-48、:206）。

**问题**

键盘和读屏用户在铃铛上被卡住：打开后无法用 Esc 退出，只能盲目 Tab 出去或用鼠标点空白处；同一头部相邻两个下拉一个可键盘操作一个不行，行为不可预测。移动端点开面板后跳到 /notifications 以外的链接（如条目跳课程页）时面板残留的风险也存在。

**建议改法**

抽 usePopoverDismiss({ triggerRef, panelRef, open })：Escape 关闭并把焦点还给铃铛按钮 + 点击外部关闭 + watch(route) 自动关闭，NotificationBell 与 AppUserMenu 共用。条目列表补 role="list"/listitem 语义。不要在面板打开时无条件把焦点移到第一条通知——该 popover 主要由鼠标点击触发，自动移焦属于焦点劫持；如需键盘可达，仅在按 ArrowDown/Enter 打开时移焦。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 AppUserMenu 已有的这套逻辑抽成 usePopoverDismiss({ triggerRef, panelRef, open })（Escape 关闭 + 焦点归还 trigger + 点击外部关闭 + 路由变化自动关闭），NotificationBell 与 AppUserMenu 共用；面板打开时把焦点移到第一条通知，条目列表加 role="list"/listitem 或 menu/menuitem 并支持上下键；useNotificationBellController 里 watch(route) 关闭面板。

</details>

<details><summary>核验记录</summary>

事实属实（行号偏移：handleDocumentClick 实为 useNotificationBellController.ts:93-102，showPanel=false 实为 :77，文件共 125 行）：NotificationBell.vue:88-96 只注册 document click，全文无任何 keydown，Esc 关不掉面板；面板容器 :22-28 是 role="region"；NotificationItem.vue:2-18 是无 role 的普通 button；controller 无 watch(route)，路由变化不重置 showPanel。对照 AppUserMenu.vue:255-258/:275-277 的 Escape + 焦点归还、:40/:48 的 role=menu/menuitem、:261-281 的上下键 roving，两者标准确实不一致。两点修正：(1) 面板可被点击外部或再次点铃铛关闭，键盘用户也能 Tab 进/出（面板在 DOM 中紧随按钮），并非「被卡住」，缺的是 Esc 这一约定动作，P2 更合适；(2) 方案里「面板打开时把焦点移到第一条通知」对鼠标点击触发的非模态 popover 是反模式（会劫持鼠标用户的焦点），应只在键盘触发时移焦，或不移焦。

</details>

---

#### `P2` 顶栏搜索无结果时显示「暂无评分数据」——文案取自图表命名空间，且没有任何后续出口

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/common/InlineSearch.vue`
- `clients/web/src/i18n/locales/zh-CN/review.ts`

**现状**

InlineSearch.vue:47-49 无结果分支渲染 `{{ t('review.chart.emptyTitle') }}`，而 zh-CN/review.ts:140 该 key 的值是「暂无评分数据」（属于 `chart` 图表命名空间，:136-143）。SearchPage.vue:229-235 的无结果态则是 `SearchX` 图标 + `t('review.search.noResults')`，两处文案体系完全不同。

**问题**

学生搜「高等数学」拼错一个字，得到的回复是「暂无评分数据」—— 答非所问，会让人以为是这门课没评分而不是没搜到课。而且无结果时下拉里没有任何补救动作：不能改用高级搜索、不给拼音/近似词建议、也不给「没找到？去发布测评」的出口，用户直接卡在这里。

**建议改法**

新增 `review.topBar.noResults`（如「没有找到相关课程」）替换 :48；并在无结果分支追加两条可点条目：「按『{query}』进行高级搜索 →」跳 `/search?courseName=`，以及「找不到课程？反馈补录」。SearchPage.vue:229 的无结果态同样补一个「修改搜索条件」按钮直接调 `backToForm()`，省掉用户滚回顶部找返回键。

<details><summary>核验记录</summary>

逐字核对无误。InlineSearch.vue:47-49 无结果分支为 `<div v-else-if="results.length === 0" class="py-6 text-center text-sm text-text-muted">{{ t('review.chart.emptyTitle') }}</div>`；i18n/locales/zh-CN/review.ts 的 chart 命名空间正好是 :136-143，其中 :140 `emptyTitle: '暂无评分数据'`（:141 emptyHint 是「成为第一个评价者吧」、:142 radarAria 是雷达图 aria，确属图表语义）。SearchPage.vue:229-235 的无结果态确为 `SearchX :size="48"` + `t('review.search.noResults')`（zh-CN/review.ts:390「未找到任何符合条件的结果，请尝试其他搜索条件」），两套文案体系不一致成立。「无结果时没有任何补救出口」也属实：通读 InlineSearch.vue 模板 1-94 行，无结果分支只有一行文字，下拉里除 :51-67 的结果项与 :74-90 的最近搜索外没有任何 footer 动作项；SearchPage.vue:229-235 同样只有图标 + 一行字，无按钮。方案落地条件齐备：review.topBar 命名空间已存在（zh-CN/review.ts:197-202），新增 noResults key 即可；SearchPage 的 backToForm() 已定义在 :462 并被 :199 的返回按钮调用，无结果态直接复用该函数无副作用。P2 定级恰当（文案错位与体验断点，不阻塞主流程）。

</details>

---

#### `P2` 顶栏搜索框只在 `/courses` 精确路径出现，课程详情/教师主页/资源页全无搜索入口

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/layout/AppHeader.vue`
- `clients/web/src/components/common/InlineSearch.vue`

**现状**

AppHeader.vue:236 `const showCourseSearch = computed(() => route.path === "/courses")`，是全等判断；InlineSearch 在 AppHeader.vue:73（桌面）和 :183（移动端）都由它 gate。对比同文件 :239 的另一个 computed 用的是 `route.path.startsWith("/courses") || route.path.startsWith("/teachers")`。

**问题**

`/courses/list`、`/courses/:id`、`/courses/:id/reviews`、`/teachers/:id`、资源页统统没有搜索框。用户在某门课的详情页想搜下一门课，唯一办法是先点回 `/courses` 首页，路径凭空多两跳。InlineSearch 还在 onMounted 注册了全局 `/` 快捷键（:270-282）并在框内渲染 `<kbd>/</kbd>` 提示，等于这个快捷键在全站 95% 的页面上是失效的。

**建议改法**

把 AppHeader.vue:236 改为 `computed(() => (route.path.startsWith('/courses') || route.path.startsWith('/teachers')) && route.name !== 'course-hub')` —— 既覆盖 /courses/list、/courses/:id、/courses/:id/reviews、/teachers/:id，又避免在 /courses hub 上与 TeachingHubPage.vue:24-48 的 hero 搜索框重复出现两个搜索入口；:124-132 移动菜单的 top 偏移沿用同一 computed 无需改。若要覆盖资源页，另外把 '/resources' 加入前缀列表。另建议给 CommandPalette 加一个可见触发按钮（调用 useCommandPalette().open）以补齐鼠标用户的全局搜索路径。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 :236 改成与 :239 一致的前缀判断（`route.path.startsWith('/courses') || route.path.startsWith('/teachers')`），或直接常驻顶栏、只在首页/登录页隐藏。若担心详情页顶栏拥挤，移动端保留折叠图标点击展开即可。

</details>

<details><summary>核验记录</summary>

行号与代码逐字吻合：AppHeader.vue:236 `const showCourseSearch = computed(() => route.path === "/courses");`，:237-239 的 showWriteReview 确实用 `route.path.startsWith("/courses") || route.path.startsWith("/teachers")`；InlineSearch 由它 gate 于 :69-73（桌面）与 :182-183（移动端）。但两处论据被夸大：① 「全无搜索入口」不准确 —— CommandPalette 由 AppShell.vue:18 全局挂载，CommandPalette.vue:219-262 同时搜课程与教师，只是仅有 Cmd/Ctrl+K 触发（useCommandPalette.ts:37-41）而无可见按钮；且顶栏导航常驻 /courses 链接（AppHeader.vue:260-265），回退成本是一次点击。② 「`/` 快捷键在全站 95% 页面失效」是空论：该监听器由 InlineSearch 自己在 :286-288 注册、:290-291 卸载，作用域与搜索框完全一致，不存在「渲染了提示但快捷键是死的」这种矛盾。再者，原方案的前缀判断会在 /courses 上留下两个搜索框：TeachingHubPage.vue:24-48 该页本身就有大号 hero 搜索输入框 + 下拉。综合为可见性/顺手度打磨项，P1 偏高，定 P2。（AppHeader.test.ts:49-54 只 stub 了 InlineSearch，无可见性断言，改动不会破测试。）

</details>

---

### 评课与互动链路

共 26 条：P0 2 / P1 12 / P2 12

#### `P0` Review.isOwner 在列表 payload 解析时被丢掉，导致自己的评课在任何列表页都没有编辑/删除入口，还能「举报自己」

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/modules/review/reviewListPayload.ts:136`
- `clients/web/src/components/business/review/ReviewCard.vue:427`
- `clients/web/src/components/business/review/ReviewCard.vue:173`
- `clients/web/src/components/business/review/ReviewFeed.vue:39`
- `clients/shared/src/types/api.gen.ts:3801`

**现状**

API 契约里 Review 有 `isOwner?: boolean`（注释：当前已认证用户是否为该测评作者），但 readReviewPayload 从头到尾没有解析这个字段（文件里连 readBoolean 都没有）。getLatestReviewsPage / getReviewsPage / searchReviewsPage 三个列表接口都走这个 reader，所以 review.isOwner 恒为 undefined。ReviewCard 的 `isOwn = props.isOwnReview ?? (review.isOwner === true)` 因此恒为 false，只有 MyReviewsTab 显式传 `:is-own-review="true"` 才是 true。

**问题**

发布成功后 router.push 跳到 /courses/:id/reviews，用户看着自己刚发的评课，卡片上没有编辑、没有删除，反而挂着「举报」按钮（v-if="!isOwn"）——用户可以举报自己的评课。想改错别字必须自己摸到「用户中心 → 我的测评」。搜索页、信息流、我的点赞页同理。

**建议改法**

1) readReviewPayload 增加 `isOwner` 的可选布尔解析；2) 同步给 ReviewFeed.vue:39、SearchPage.vue:332、MyVotesTab.vue:24 的 <ReviewCard> 补上 `@deleted` / `@updated` 监听（目前只有 MyReviewsTab 有），否则修好 isOwner 后会立刻暴露「删除成功但卡片还在、编辑成功但正文没变」的问题；3) 举报按钮改成 `v-if="!isOwn && isAuthenticated"`。

---

#### `P0` 发布表单校验不通过时提交按钮「点了没反应」，没有任何提示也不定位到出错字段

> **已逐行复验**（报告人）　|　工作量：M

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:1003`
- `clients/web/src/modules/review/views/PostReviewPage.vue:1018`
- `clients/web/src/modules/review/views/PostReviewPage.vue:992`
- `clients/web/src/modules/review/views/PostReviewPage.vue:406`

**现状**

handleSubmit 里先 showErrors.value = true，然后 `if (!canSubmit.value || !course) return` 直接静默返回：不弹 toast、不写 submitError、不滚动也不聚焦到第一个出错字段。提交按钮 `:disabled="submitting"` 只在提交中禁用，表单没填完时按钮外观和可点性完全正常。canSubmit 依赖 course + termID + 全部评分维度 + title + content 五组条件。

**问题**

表单是 6 + N 个控件（N 是后端返回的评分维度数），提交按钮在最底部。用户最常见的失误是漏打某个评分维度——错误提示只出现在评分区那一行（离按钮 600-900px），提交按钮附近的 submitError 区域是空的。用户点提交后页面没有任何变化，只会反复点击然后放弃。这是整个产品最重要的转化路径上的死结。

**建议改法**

1) `if (!canSubmit.value || !course)` 分支里补 `submitError.value = t('review.postForm.errors.incomplete')` + toast.error，让按钮上方那块红色提示区（第 399-405 行）真正用起来；2) 提交失败后 `await nextTick()` 再 `document.querySelector('[aria-invalid="true"]')?.scrollIntoView({ block: 'center' })` 并 focus；3) 在错误提示块里列出未完成项（课程/学期/评分/标题/正文）并支持点击跳转。

---

#### `P1` 内容预检命中敏感词时提示「请检查后提交」，但代码紧接着就把评课发出去了

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:1032`
- `clients/web/src/i18n/locales/zh-CN/review.ts:98`

**现状**

checkContentResult 返回 isValid=false 且 level==='warn' 时，只 `toast.warning(t('review.post.contentWarning'))`（文案：内容可能含有敏感词汇，请检查后提交），然后代码继续往下执行 createReview，成功后再 `toast.success` 并 router.push 跳走。

**问题**

提示语明确要求用户「检查后提交」，实际上已经提交完了。用户会看到黄色警告和绿色成功两个 toast 同时叠在右上角、页面还跳走了，完全不知道这条评课到底发出去没有、要不要重发。

**建议改法**

二选一：(a) 文案改成「已发布，内容命中敏感词提醒，可能会被人工复核」并保持直接发布；(b) 真的拦一下——弹一个确认框列出问题并给「仍然发布 / 返回修改」，确认后再调 createReview。推荐 (b)，项目里已有现成的 ContentQualityTip.vue 组件（目前无人使用）可以直接用。

---

#### `P1` 发布表单两处文案错误：教师下拉的空选项写着「暂无教师数据」，英文默认模板显示字面量 \n

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:175`
- `clients/web/src/i18n/locales/zh-CN/review.ts:105`
- `clients/web/src/i18n/locales/en-US/review.ts:354`
- `clients/web/src/modules/review/views/PostReviewPage.vue:520`

**现状**

1) 教师 select 的空选项文案是 `selectedCourse ? t('review.post.teacherNone') : t('review.postForm.selectCourse')`，而 teacherNone 就是「暂无教师数据」/「No teacher data」；2) en-US 的 `defaultTemplate` 写成 'Course experience:\\nAssignments/workload:\\nAbout exams:'（双反斜杠），zh-CN 是真实换行；这个值直接赋给 content 作为正文初始内容。

**问题**

1) 选了一门有 3 位老师的课后，下拉第一项赫然写着「暂无教师数据」，下面却列着 3 位老师，而标签旁边又挂着「选填」徽章，用户完全不知道该不该选、能不能不选；2) 英文用户打开发布页，正文框里是一行 `Course experience:\nAssignments/workload:\nAbout exams:`，字面的反斜杠 n 直接暴露给用户，还得手动删掉换行。

**建议改法**

1) 空选项文案新增 `review.post.teacherUnspecified`（不指定教师）用于「有教师但不选」，只有 teachers.length === 0 时才用 teacherNone 并同时 disable select；2) en-US 的 defaultTemplate 把 \\n 改成 \n；3) 增加一条 i18n 单测断言 zh/en 的 defaultTemplate 行数一致。

<details><summary>核验记录</summary>

两处事实全部核对无误。PostReviewPage.vue:175-177 空选项确为 `{{ selectedCourse ? t('review.post.teacherNone') : t('review.postForm.selectCourse') }}`，而 zh-CN/review.ts:105 teacherNone='暂无教师数据'、en-US/review.ts:120='No teacher data'；同一 label 上（PostReviewPage.vue:161-163）确实挂着 review.post.teacherOptional（'选填'/'Optional'）徽章，select 只在 `:disabled="!selectedCourse"`（:173）时禁用，教师列表非空时空选项仍显示「暂无教师数据」。第二处：en-US/review.ts:354 源码字面量为 'Course experience:\\nAssignments/workload:\\nAbout exams:'（双反斜杠 → JS 字符串里是反斜杠+n），zh-CN/review.ts:354 用的是真实转义换行 '课程听感：\n作业/任务量：\n关于考试：'；该值经 PostReviewPage.vue:494 的 computed 直接赋给 content（:520，另见 :836 恢复草稿、:801/:977 的模板比对），vue-i18n 消息编译器不处理纯文本段中的反斜杠转义，故英文用户 textarea 里会出现字面 \n。方案可行：teacherNone 目前只被这一处引用，新增 teacherUnspecified 键不影响他处；ratingDisplayPolicy 等测试均未断言 defaultTemplate（全仓 grep 只有 locales 与 PostReviewPage 引用），新增行数一致性断言无冲突。

</details>

---

#### `P1` 发布表单的评分行在 375px 以下手机上横向溢出，最高分按钮被挤出屏幕（评分是必填项）

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:249`
- `clients/web/src/components/business/review/EmojiRatingInput.vue:2`
- `clients/web/src/components/business/review/EmojiRatingInput.vue:10`

**现状**

每个评分维度是一行 `flex items-center justify-between`：左边维度名 `shrink-0`，右边 EmojiRatingInput 是 `flex items-center gap-1`（无 flex-wrap）里的 5 个按钮，每个按钮 `p-2 m-0.5 border-2` 包一个 `size-6` 的 svg = 24+16+4+4 = 48px 实占宽，5 个共 240px 加 4 个 gap-1 共 256px，且都不可压缩。

**问题**

375px 视口下可用宽度 = 375 − 32(页面 px-4) − 48(卡片 p-6) = 295px，加 4 个汉字的维度名（约 56px）已经是 312px，直接溢出；320-360px 的机型上第 5 档（最高分）整个跑到屏幕外。英文界面下维度名更长（Assessment & Grading），溢出更严重。评分是必填项，配合上一条「提交无反馈」，移动端用户会彻底卡住。

**建议改法**

保留原方案的技术路线（可行）：PostReviewPage.vue:249-253 的行改成 `flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-between`，EmojiRatingInput.vue:2 根节点加 flex-wrap（在 items-start 的列容器里根节点按 fit-content 收敛到可用宽，wrap 才真正生效），并对按钮做窄屏降级 `p-1.5 size-5`（实占 40px，5 个 236px < 320px 屏的 240px 可用宽）。补一条原方案漏掉的：EmojiRatingInput.vue:23-28 的 error span 与 5 个按钮同处一行，提交校验失败时会再挤进 ~100px（如「请选择工作量评分」），是溢出最严重的时刻，应把该 span 移出按钮行、独立成块级一行。验收按 320/360/375/390px 四档量，并确认 document.documentElement.scrollWidth === clientWidth。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

评分行改成窄屏纵向：`flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-between`，并给 EmojiRatingInput 根节点加 `flex-wrap`；同时窄屏把按钮降到 `p-1.5` + `size-5`（实占 40px，5 个 200px），保证 320px 屏也能整行放下。

</details>

<details><summary>核验记录</summary>

事实核对基本准确。PostReviewPage.vue:249-264 每个维度行确为 `flex items-center justify-between`，左侧 span 带 shrink-0（:254），右侧 EmojiRatingInput（:257）根节点 EmojiRatingInput.vue:2 是 `flex items-center gap-1` 无 flex-wrap，按钮 :10 `p-2 m-0.5 border-2` 包 :19 的 `size-6` svg。宽度算式我逐项验证过：styles/tailwind.css 的 @theme（11-140 行）只覆盖了 --radius-*（79-84），没有覆盖 --spacing/--text-sm，所以 p-2=8px、m-0.5=2px、gap-1=4px、text-sm=14px；虽然 tailwind.css:435-440 的 base 里有 `button { border: none }`，但 Tailwind v4 的 border-width 工具类同时输出 `border-style: var(--tw-border-style)`（node_modules/tailwindcss/dist/lib.js 中 g("border",{width:k=>[o("border-style","var(--tw-border-style)"),o("border-width",k)]}) 已确认），utilities 层压过 base，所以 2px 边框真实存在 → 单按钮实占 24+16+4+4=48px，5 个 + 4 个 gap = 256px，且按钮 min-width:auto 下不可压缩。容器可用宽 = 375 − 32(:2 px-4) − 48(:21 p-6) = 295px，4 字中文维度名（课程难度/教学质量/内容质量，见 i18n/locales/zh-CN/review.ts:222-232 与 ratingHelpers.ts:21-29 的映射）约 56px，合计 312px，确实溢出。
但 P0 站不住：(1) 全仓没有任何 overflow-x:hidden（styles/ 下 grep overflow 无结果，tailwind.css:362-473 base、:503-506 .app-shell-main、AppShell.vue 模板都没有裁剪），页面会产生横向滚动，第 5 档按钮仍可横滑点到；(2) 「320-360px 整个跑到屏幕外」不成立——360px 时溢出仅 ~32px，48px 宽的第 5 档只是被截去 2/3，要到 ≤344px 视口才整体出屏；(3) 论证里依赖的「上一条提交无反馈」不成立，handleSubmit（:1003-1018）会置 showErrors=true 触发行内错误、失败分支还有 toast + submitError。综合是严重的移动端布局缺陷但非阻断性，应为 P1。

</details>

---

#### `P1` 发布表单的错误红框永远不显示：只写了 border-danger 没有 border 宽度，Tailwind v4 preflight 把边框重置成了 0

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:68`
- `clients/web/src/modules/review/views/PostReviewPage.vue:198`
- `clients/web/src/modules/review/views/PostReviewPage.vue:286`
- `clients/web/src/modules/review/views/PostReviewPage.vue:326`
- `clients/web/src/styles/tailwind.css:563`

**现状**

课程搜索框、学期下拉、标题输入、正文文本域的 class 都是 `bg-bg-elevated ... focus:border-primary focus:ring-2 focus:ring-primary/20`，没有任何 border 宽度类；错误态绑的是 `:class="{ 'border-danger focus:border-danger focus:ring-danger/20': showErrors && !title.trim() }"`。Tailwind v4 的 preflight 是 `*{ border: 0 solid }`，所以这些类只改 border-color，宽度是 0 → 边框不可见。项目里已有 `.focus-ring-field` 工具类（ReplyForm 在用），发布页没用。

**问题**

校验失败时用户期望看到的红框完全不出现，唯一线索是下面一行 12px 的红字；再加上 bg-bg-elevated(#f4f2f8) 压在 bg-bg-card(#ffffff) 上对比度只有约 4%，浅色主题下连「哪里是输入框」都看不清楚。这直接放大了第 1 条「提交没反应」的问题。

**建议改法**

统一改用 `.focus-ring-field` 并补 `border border-border`，错误态用 `!border-danger`；把这套 field 样式抽成一个 @utility（如 `.field-input` / `.field-input-error`）放进 tailwind.css，发布页、内联编辑、管理弹窗共用。

---

#### `P1` 审核/管理员改稿弹窗的输入框用了不存在的 bg-bg-input，实际渲染成完全透明无边框

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/components/business/review/ModerationDialog.vue:34`
- `clients/web/src/components/business/review/AdminEditDialog.vue:33`
- `clients/web/src/components/business/review/AdminEditDialog.vue:46`
- `clients/web/src/components/business/review/AdminEditDialog.vue:60`
- `clients/web/src/components/business/review/ReplyForm.vue:20`

**现状**

这 4 个输入控件的 class 是 `rounded-lg bg-bg-input px-3 py-2 ... focus:ring-2`，但 tailwind.css 的 @theme 里没有 --color-bg-input（只有 bg-base/card/elevated/hover/secondary/tertiary/overlay/glass），main.css 也没有兜底的 input 基础样式，所以这个类不生成任何 CSS，且这些控件也没有 border 类。同类问题：ReplyForm.vue:20 的 `text-color-rating-3`（正确的是 text-rating-3）、ReplyCard.vue:32 的 `bg-bg-primary`（不存在）。

**问题**

「屏蔽测评」和「管理员改稿」这两个低频但高风险的弹窗里，输入框和卡片背景完全融为一体，管理员看不出哪里能输入、边界在哪，只有点进去出现 focus ring 才知道；改稿弹窗里三个输入框叠在一起更是分不清哪块是标题、哪块是正文。

**建议改法**

三处失效类名改成现有 token：`bg-bg-elevated border border-border`（或按需在 tailwind.css 的 @theme 和 design-system/tokens.ts 同步补一个 --color-bg-input）；顺手把 text-color-rating-3 → text-rating-3、bg-bg-primary → bg-bg-card。建议加一条 lint/单测扫描模板里 `bg-bg-*` / `text-*` 是否都能在 @theme 里找到对应 token。

---

#### `P1` 未定义的 token 类名静默渲染成透明：除已知的 bg-bg-input 外还有 bg-bg-primary

> **部分复验**（核心事实已确认，附带断言由 agent 核验）　|　工作量：S

**位置**

- `clients/web/src/components/business/review/ReplyCard.vue`
- `clients/web/src/components/business/review/AdminEditDialog.vue`

**现状**

按 tailwind.css @theme 里实际定义的 43 个 --color-* 逐条比对全仓库用到的 71 个 token 工具类，除已知的 bg-bg-input（AdminEditDialog.vue:33/46/60、ModerationDialog.vue:34）外，还有一处未定义：ReplyCard.vue:32 的 class="rounded-sm border border-border bg-bg-primary px-3 py-1 text-xs text-text-secondary ..."——@theme 里有 --color-bg-base / bg-card / bg-elevated / bg-hover / bg-secondary / bg-tertiary，唯独没有 --color-bg-primary。此外还有一类语义陷阱：bg-secondary / text-secondary 命中的是 --color-secondary（#5ab8cc 品牌青色），不是文字次级色 --color-text-secondary，两者只差一个前缀。

**问题**

Tailwind v4 对未定义 token 不生成任何类，也不报错、不警告，构建照常通过。ReplyCard 那个回复操作按钮因此完全没有背景色，只剩一圈 rgba(0,0,0,0.06) 的极淡边框（--color-border 本身就设计成『极淡分隔线』级别），在 bg-bg-card 白底上几乎看不见是个按钮；暗色下更糟，border 变成 rgba(255,255,255,0.06)。同理 bg-bg-input 让四个后台/审核弹窗的输入框失去背景。这类错误只能靠人眼在特定页面撞见，是持续复发的一类缺陷。

**建议改法**

①ReplyCard.vue:32 改成 bg-bg-elevated（和其他次级按钮一致），bg-bg-input 的 4 处统一改成 bg-bg-elevated 并补 border border-border（同第 3 条）。②补一条 CI 护栏堵住整类问题：写个脚本从 tailwind.css 的 @theme 块解析出所有 --color-*/--radius-*/--shadow-*/--z-*，再扫 .vue 里所有形如 (bg|text|border|ring|from|to|via|fill|stroke|divide|placeholder|shadow)-(bg|text|border|rating|vote)-* 的类名做集合差集，差集非空就 fail。③把 --color-secondary 更名为 --color-brand-teal（或给文字色统一加 fg- 前缀），消除 text-secondary 与 text-text-secondary 的歧义。

---

#### `P1` 未登录用户在信息流里看不到任何点赞/回复数，点「立即登录」还不带 redirect，登录后回不到原来的评测

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/components/business/review/ReviewCard.vue:432`
- `clients/web/src/components/business/review/ReviewCard.vue:331`
- `clients/web/src/components/business/review/ReviewCard.vue:545`
- `clients/web/src/modules/review/views/CourseDetailPage.vue:840`
- `clients/web/src/modules/review/useReviewVoting.ts:43`

**现状**

ReviewCard 的 `showActions = isAuthenticated && !isHidden`，未登录时整个操作栏（点赞/点踩/评论计数/举报）都不渲染；正文又被 LockedReviewContent 替换掉，于是 toggleExpand 完全不可达 —— 卡片里那段 `ReplyLoginPrompt v-if="!isAuthenticated"`（第 331 行）是死分支。而 CourseDetailPage 对未登录用户照常显示计数，点击时 useReviewVoting 走 `router.push({ name:'login', query:{ redirect: fullPath } })`。同时 ReviewCard.vue:546 和 CourseDetailPage.vue:841 调的是 `authStore.login()`，一个参数都没传，而 authStore.login(redirect) 是支持 redirect 的（JoinStartPage 就传了）。

**问题**

1) 主信息流对未登录访客隐藏了「这条有多少人赞、有多少讨论」这种最强的转化钩子，两个页面对同一状态的表现还不一致；2) 用户在某条评测上点「立即登录」，走完 OIDC 回来落在默认页，刚才在看的那条要自己重新找；3) 项目里同时存在「跳站内 login 路由」和「直接发起 OIDC」两套登录入口。

**建议改法**

1) 未登录时也渲染只读的点赞/回复计数（按钮 aria-disabled="true"，点击走登录），并让评论按钮可展开以露出 ReplyLoginPrompt；2) 两处改成 `authStore.login(route.fullPath)`；3) 统一登录入口：要么都用 `router.push({name:'login', query:{redirect}})`（FavoriteButton、useReviewVoting 已经这么做），要么都用 authStore.login(redirect)。

<details><summary>核验记录</summary>

三处论断逐一验证通过。(1) ReviewCard.vue:432 `showActions = isAuthenticated && !isHidden`，操作栏 :130-210 整体 v-if=showActions，未登录不渲染；正文 :83-89 被 LockedReviewContent 顶掉，于是 toggleExpand 的两个触发点（:110-112 的正文点击、:166 的评论按钮）在未登录时都不存在，isExpanded 永远为 false，:309 的展开区不渲染，所以 :331-334 的 `ReplyLoginPrompt v-if="!isAuthenticated"` 确实是死分支（canManageReviews 依赖 authStore.user，未登录恒 false，hidden 分支也救不回来）。信息流入口 ReviewFeed.vue:39-46 只渲染 ReviewCard，未登录访客确实一个计数都看不到。(2) CourseDetailPage.vue:336-374 的点赞/点踩/评论计数没有任何登录门槛，点击进 useReviewVoting.ts:43-53 → `router.push({name:'login', query:{redirect: fullPath}})`，两个页面对同一状态表现不一致属实。(3) ReviewCard.vue:545-547 是 `authStore.login()`、CourseDetailPage.vue:840-842 是 `void authStore.login()`，均未传参；stores/auth.ts:404 `login = (redirect?: string) => ...`，:333-337 在 redirect 为空时压根不调用 resolvePostLoginRedirectTarget，直接把 undefined 发给后端，而 server/internal/modules/auth/handler_login.go:692-698 resolveRedirectTarget 在 redirect 为空时回落 defaultRedirectURL —— 「回来落在默认页」成立；JoinStartPage.vue:243 `auth.login(currentReturnURL.value)` 证明传参是支持的。方案可行：utils/redirect.ts:46-48/76-80 的 isSafeRelativeRedirect 允许 `/` 开头的相对路径，`authStore.login(route.fullPath)` 会被 resolvePostLoginRedirectTarget 正常转成同源绝对地址。补一点实施注意：若按方案给未登录用户渲染可点计数，必须让点击走登录分支——ReviewCard 现用的 useReviewVote.ts:52 没有任何鉴权判断，直接放开会打出必然 401 的投票请求。

</details>

---

#### `P1` 草稿恢复弹窗关不掉：Esc 和点遮罩都无效，唯一的退出选项是「放弃草稿」（立即删除且不可撤销）

> **已逐行复验**（报告人）　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:421`
- `clients/web/src/components/business/review/DraftPromptDialog.vue:80`
- `clients/web/src/components/business/review/DraftPromptDialog.vue:26`

**现状**

恢复草稿的 DraftPromptDialog 只绑定了 @confirm 和 @discard。组件内部 dismiss()（Esc 键经 useDialogFocus、点击遮罩 @click.self）emit 的是 'keep'，这里没有监听 → 弹窗不会关闭。同时没传 cancelText，`v-if="cancelText"` 的取消按钮压根不渲染，弹窗只有 [放弃草稿][载入草稿] 两个按钮。对比下方的离开确认弹窗（第 433 行）是绑了 @keep 的。

**问题**

用户从课程页点「发布测评」进来，第一屏就被一个关不掉的模态挡住；如果他这次想写另一门课的新评测，界面上唯一看起来像「跳过」的按钮是「放弃草稿」，而它会立刻 deleteDraft() 删掉服务端草稿，没有二次确认也没有撤销。

**建议改法**

给恢复弹窗传 `cancel-text="稍后再说"` 并绑定 `@keep="restorePromptDraft = null"`，让 Esc/遮罩/取消都只是关闭弹窗、保留草稿；「放弃草稿」加二次确认，或干脆从这个弹窗里去掉，把删除动作只留在 DraftIndicator 上。

---

#### `P1` 评分维度接口失败后没有重试入口，用户写完的内容只能靠刷新整页自救

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/PostReviewPage.vue:242`
- `clients/web/src/modules/review/views/PostReviewPage.vue:504`
- `clients/web/src/modules/review/views/PostReviewPage.vue:1006`
- `clients/web/src/components/business/review/composables/useRatingDimensions.ts:75`

**现状**

useRatingDimensions 已经 return 了 load()，但 PostReviewPage 只解构了 dimensions / loading / loadFailed。加载失败时只渲染一行红字「评分维度加载失败，请稍后重试」，没有任何按钮；此时点提交，handleSubmit 只是把同一句话塞进 submitError 再 return。`ratingDimensions.length === 0` 的分支（「暂无可用评分维度」）同样是死路，因为 areRatingsComplete 在维度为空时恒返回 false。

**问题**

用户可能已经写完标题和几百字正文，却因为一个后台接口抖动而永远无法发布，唯一出路是刷新整页（还要再过一遍草稿恢复弹窗）。文案说「请稍后重试」，但界面上没有「重试」这个东西。

**建议改法**

解构出 load，在失败提示右侧加「重试」按钮直接调用 load()；维度为空时同样给出重试 + 反馈入口（复用课程搜索空态里的联系方式），并让提交按钮上方的错误块说明原因。

<details><summary>核验记录</summary>

逐行核实无误：useRatingDimensions.ts:75-101 确实 return 了 load()，但 PostReviewPage.vue:504-508 只解构 dimensions/loading/loadFailed，没拿 load；模板 :242-244 失败态只有一行红字 `t('review.post.ratingLoadFailed')`，我在该 fieldset（:235-266）内 grep 不到任何 button/retry（对比 :415-428 的 CourseDetailPage 和 ReviewCard.vue:313-318 都是有 retry 按钮的，说明这里是漏掉而非风格如此）。文案确实写着「请稍后重试」（i18n/locales/zh-CN/review.ts:109）却没有重试控件。handleSubmit（:1006-1010）在 loadFailed 时只是把同一句塞进 submitError + toast 后 return。维度为空（:245-247）更糟：areRatingsComplete（useRatingDimensions.ts:103-108）在 dimensions.length===0 时恒 false → allRatingsProvided(:986-990) false → canSubmit(:992-1000) false → handleSubmit 在 :1018 静默 return，连 submitError 都不给，是彻底的死路。load() 只在 onMounted(:89-91) 跑一次，语言切换/重选课程都不会重发，唯一出路确实是刷新整页，而刷新后 onMounted(:750) 的 promptDraftRestoreOnEntry 会再弹一次草稿恢复模态，描述属实。方案（解构 load 并在失败提示旁加重试按钮）可行且改动极小。

</details>

---

#### `P1` 院系/分类筛选状态完全不进 URL，点进课程再返回全部重置并重新请求

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/components/business/review/DepartmentSidebar.vue`
- `clients/web/src/components/business/review/CourseListItem.vue`
- `clients/web/src/modules/review/views/ReviewPage.vue`

**现状**

DepartmentSidebar.vue:111 的 activeCategory、:127 的 expandedDepts、:130 的 deptCourses 全是组件内部 ref，没有任何 route.query 读写。selectCategory（:184-191）还会 deptCourses.clear()，loadDepartments（:166）会把 expandedDepts 清空。侧栏挂在 ReviewPage.vue:32/44，而点击 CourseListItem.vue:3 跳转的 /courses/:id 在 router/index.ts:331 是和 /courses/reviews（:323）平级的顶层路由，两者不是父子关系。

**问题**

用户的找课路径是「选分类 → 展开某个院系 → 从十几门课里挑一门」，这三步的成本全在侧栏里。但因为 ReviewPage 会随路由整体卸载，点进任意一门课再按浏览器后退，回来时分类回到「全部」、所有院系收起、课程缓存清空，getCategories + getDepartments 重新打两次，用户必须把整条路径重走一遍——而「看几门课再对比」正是这个页面最常见的用法。同时筛选结果无法分享：没法把「计算机学院」这个视图发给同学。

**建议改法**

把 category 和展开的 dept id 同步到 route.query（?category=xxx&dept=12），组件挂载时从 query 初始化 activeCategory / expandedDepts，切换时用 router.replace 写回——CourseListPage.vue:106-158 已经有现成的 query 合并写法（courseListQueryWithSearch）可以直接抽成 composable 共用。更彻底的做法是把 /courses/:id 改成 /courses/reviews 的 children 路由，让侧栏在选课时不卸载；这也顺带修好当前失效的 ReviewPage.vue:52 router-view。

<details><summary>核验记录</summary>

核实无误：DepartmentSidebar.vue:111 activeCategory、:127 expandedDepts、:130 deptCourses 全是组件内 ref，全文无一处 route.query 读写（只有 :134 读 route.params.id 做高亮）；selectCategory（:184-191）clear 缓存；loadDepartments（:166）清空 expandedDepts。侧栏挂在 ReviewPage.vue:32/44；CourseListItem.vue:3 跳 /courses/:id，router/index.ts:323 与 :331 确为平级顶层记录，且 App.vue:5-9 的 router-view 没有 KeepAlive，所以 ReviewPage 必然整体卸载、onMounted 重新打两次请求。方案可行：CourseListPage.vue:106-158（courseListQueryWithSearch + router.replace + watch route.query）确是现成范式；改成 children 路由用绝对路径 '/courses/:id' 也能让 matched.length>1（顺带救活 ReviewPage.vue:52），但需注意 PostReviewPage.vue:1060 用的是 name: 'course-reviews'。P1 合理。

</details>

---

#### `P1` 院系侧栏：分类接口失败会连带吞掉整个院系导航

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/business/review/DepartmentSidebar.vue`

**现状**

DepartmentSidebar.vue:19 的 v-if="sidebarError" 和 :31 的 v-else 是同一组分支。loadCategories() 失败时 :147-150 会写 sidebarError；而院系树整个在 v-else 分支里。onMounted（:261-264）并行发起 loadCategories() 和 loadDepartments()，loadDepartments 在 :155 会先把 sidebarError 清空再 await，所以后返回的 categories 失败一定能盖掉已经成功的 departments。

**问题**

分类 chips 只是次要筛选器，院系树才是 /courses/reviews 唯一的课程导航。现在只要 getCategories 挂了（哪怕 getDepartments 完全正常、数据就在 departments.value 里），用户看到的是：一个孤零零的「全部」chip + 一个黄色错误框，下面空空如也，整个页面失去了找课的入口。而且 :175-179 的 retrySidebar 会把两个请求一起重发，用户点重试后仍可能重复踩同一个失败点。

**建议改法**

拆成 categoryError / departmentError 两个 ref：1) loadCategories 开头 `categoryError.value = ''`，catch 里只写 categoryError，并把它渲染成 :4-17 分类 chips 行内的一条细提示 + 只重试 loadCategories 的按钮；2) loadDepartments 把 :155 的 `sidebarError.value = ''` 改成 `departmentError.value = ''`（而不是删除该行），catch 里只写 departmentError，:19 的错误块条件改为 `v-if="departmentError"`、院系树保持 `v-else`；3) :81 空状态的 gate 由 `!sidebarError` 改为 `!departmentError`；4) :175-179 retrySidebar 拆成 retryCategories / retryDepartments 两个入口，避免用户重试时重复踩同一个失败点。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

拆成 categoryError 和 departmentError 两个 ref：categoryError 只在 :4-17 的分类 chips 行内渲染成一条细提示（带只重试 loadCategories 的按钮），departmentError 才控制 :31 那块。院系树的渲染条件改为 v-else-if="!departmentError"，保证分类失败时院系导航照常可用。同时删掉 loadDepartments 里 :155 那句 sidebarError.value = ''，避免跨请求互相清状态。

</details>

<details><summary>核验记录</summary>

事实全部核对无误：DepartmentSidebar.vue:19 `v-if="sidebarError"` 与 :31 `v-else` 确为同一组分支，院系树整块在 v-else 里；:147-150 的 catch 写 sidebarError；:153-155 loadDepartments 在 await 之前同步执行 `sidebarError.value = ''`；:261-264 onMounted 先调 loadCategories()（已进入 await）再调 loadDepartments()（同步清空 error），因此清空一定早于 categories 的 catch —— getCategories 失败 + getDepartments 成功时，院系树确实被整块隐藏（:81 的空状态也被 `!sidebarError` 一起 gate 掉），:175-179 retrySidebar 会重发两个请求。跨请求状态污染是真 bug。但降级理由不成立：同页右栏 ReviewFeed 仍在渲染 ReviewCard，ReviewCard.vue:46-47 有 `router-link :to="/courses/${review.courseID}/reviews"`；此外 /courses 首屏大搜索框（TeachingHubPage.vue:24-48）、/courses/list、全局 Cmd+K 面板（AppShell.vue:18 + CommandPalette.vue:219-262）都还在，并非「整个页面失去找课入口」，且触发条件是后端部分故障、页面仍给出可点的重试。故 P0 过高，定 P1。另原方案有一处会引入新 bug：直接删掉 :155 而不在成功路径复位，departmentError 会变成粘滞状态（例如 loadDepartments 先失败、之后 selectCategory 重新加载成功，:161-166 成功分支从不清 error，院系树将继续被隐藏）。

</details>

---

#### `P1` 高级搜索的结果不写入 URL：点进课程再返回，全部搜索条件清零

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/SearchPage.vue`
- `clients/web/src/modules/course/views/CourseListPage.vue`

**现状**

SearchPage.vue:674 `handleSearch()` 只把 `showResults` 置 true 并直接请求，全程不调 `router.push`；SearchPage.vue:462 `backToForm()` 也只是把 flag 置 false。URL 始终停在 `/search`（无 query）。SearchPage.vue:737 `syncSearchFromRoute()` 在 onMounted 时读 `route.query`，读不到就走 `backToForm()`。全仓库无 keep-alive（grep `keep-alive|KeepAlive` 零命中）。

**问题**

用户填完 5 个条件搜出结果 → 点开一门课 → 浏览器返回 → 组件重新挂载 → `hydrateSearchFormFromRoute()` 从空 query 读到空值 → `backToForm()` 把表单和结果全清空，用户回到一张白表单，必须从头再填一遍。同样地搜索结果无法收藏、无法分享给同学、无法刷新。雪上加霜的是这个页面本身几乎没有入口：全站对 `/search` 的引用只有 CourseListPage.vue:301（`advancedSearchRoute`）一处，AppHeader 导航没有、CommandPalette 也不跳 `/search`。

**建议改法**

1) handleSearch 保持为纯执行函数，在表单 submit 处理器（:7 @submit.prevent）里改为校验通过后 `router.push({ name: 'search', query: buildQueryFromForm() })`，由已有的 watch(route.query)→syncSearchFromRoute 单向驱动搜索，避免双写；query 键沿用 hydrate 已支持的 courseName/courseCode/departmentID/teacherName/termID（:545-550）。2) backToForm 保持为纯状态重置函数（供 :749 内部复用），只把 :199 「返回」按钮的 handler 换成 `router.push({ name: 'search', query: {} })`，防止 replace 回环；push 到相同空 query 时 vue-router 判定为重复导航、watch 不触发，需保留按钮直接调 backToForm 作为兜底。3) 入口部分只需在 CourseListPage 之外考虑 InlineSearch 下拉补一个入口即可，TeachingHubPage.vue:114 的「高级搜索」链接已存在，不必重复添加。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

`handleSearch()` 成功校验后先 `router.push({ path: '/search', query: buildQueryFromForm() })`，把实际搜索完全交给已有的 `watch(() => route.query)` → `syncSearchFromRoute()` 单向驱动（这条链路已经写好了，只是没人往里塞 query）；`backToForm()` 改成 `router.replace({ path: '/search', query: {} })`。同时在 AppHeader 导航或 InlineSearch 下拉底部加一个「高级搜索」入口指向 `/search`。

</details>

<details><summary>核验记录</summary>

核心缺陷属实：SearchPage.vue:674-735 handleSearch 全程无 router.push，只置 showResults 并直接发请求；:462-472 backToForm 只重置本地状态；:544-559 hydrateSearchFormFromRoute 只读 route.query；:737-750 syncSearchFromRoute 读不到条件就调 backToForm；:794-798 onMounted、:800-805 watch(route.query) 链路确实已就绪只是没人写 query；全仓 grep keep-alive/KeepAlive 零命中。但有一条事实错误且是本条严重度的主要论据：「全站对 /search 的引用只有 CourseListPage.vue:301 一处」不成立 —— /courses 首页 TeachingHubPage.vue（router/index.ts:255-258）在 :304-309 定义 `advancedSearchRoute`（`{ name: 'search', query: { courseName } }`），:114-115 有可见入口链接，:317-331 的 Enter 无选中项时还会 `router.push(advancedSearchRoute)`。也就是说主入口已经把 courseName 写进 URL，从 hub 进来的场景返回时是能恢复该条件的，真正丢失的是用户在页内追加的院系/教师/学期等条件。综合：无数据丢失、主路径部分可恢复、入口并不稀缺，P0 过高，定 P1。另原方案的 backToForm 改造有再入风险：syncSearchFromRoute:749 自身会调用 backToForm，若在其中做 router.replace 会形成 query 变更→watch→syncSearchFromRoute→replace 的回环。

</details>

---

#### `P2` 5 个评分/统计组件是彻底的死代码，同一根评分进度条却被手搓了 3 遍

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/components/business/review/TeacherStatsCard.vue`
- `clients/web/src/components/business/review/SemesterStatsGrid.vue`
- `clients/web/src/components/business/review/RatingDistribution.vue`
- `clients/web/src/components/business/review/RatingDisplay.vue`
- `clients/web/src/components/business/review/DimensionBars.vue`
- `clients/web/src/components/common/RatingBar.vue`
- `clients/web/src/modules/review/views/CourseDetailPage.vue`

**现状**

全仓库（排除 __tests__ 与自动生成的 components.d.ts）对 `<TeacherStatsCard` / `<SemesterStatsGrid` / `<RatingDistribution` / `<RatingDisplay` / `<DimensionBars` 及其 kebab 形式的引用数均为 0。RatingBar.vue 唯一的消费者是同样已死的 DimensionBars.vue:3。与此同时 CourseDetailPage.vue:132-147、SemesterStatsGrid.vue:31-45、RatingDistribution.vue:10-19 各自内联复写了同一根 `h-2 bg-bg-secondary rounded-full overflow-hidden` + 彩色填充条。真正在用的只有 EmojiRating（ReviewCard、CourseDetailPage、TeacherHubPage、TeacherProfilePage、TeacherStatsCard）和 RatingCircle（仅 TeacherProfilePage:29 一处）。

**问题**

约 320 行组件代码 + 对应的 i18n key（`review.stats.*`、`teaching.profile.avgRating` 等）常年参与打包与类型检查却永不渲染；新人改评分展示时会改到没人看的文件。同时 CourseDetailPage 的维度条和 RatingBar 的样式已经漂移（label 宽度 `w-16` vs `min-w-[60px]`、过渡 `duration-slow` vs `duration-700`），未来接入 SemesterStatsGrid 会直接出现第三种视觉。

**建议改法**

删除 RatingDisplay.vue / RatingDistribution.vue / DimensionBars.vue / TeacherStatsCard.vue / SemesterStatsGrid.vue，同一 PR 内必须同步改 clients/web/src/modules/review/__tests__/ratingDisplayPolicy.test.ts：移除 policySources 中的 ratingDisplay/ratingDistribution/semesterStatsGrid/teacherStatsCard 四项（:14-17）及 :38-39 两条断言，其余对 ReviewCard/EmojiRating/CourseDetailPage/TeacherProfilePage/TeacherHubPage/RatingBar/RatingCircle 的策略断言保留。保留 RatingBar.vue 并把 CourseDetailPage.vue:121-148 的内联维度条换成 RatingBar（给它加 ariaLabel prop，把 dimensionRatingBarAriaLabel 的调用传进去；注意 ratingDisplayPolicy.test.ts:50-56 仍断言 CourseDetailPage 源码含 role="img"、:aria-label="dimensionRatingBarAriaLabel(dim)" 与 aria-hidden="true"，重构时要么保持这些字符串在 CourseDetailPage 内，要么同步改断言）。顺带删掉只被 SemesterStatsGrid 使用的 review.stats.insufficient / reviewCount / insufficientHint 三个 key（zh-CN 与 en-US 同步），review.stats.noData 必须保留。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

删掉 RatingDisplay.vue、RatingDistribution.vue、DimensionBars.vue、TeacherStatsCard.vue、SemesterStatsGrid.vue 五个文件（TeacherStatsCard 的卡片形态若还想要，应该直接替换 TeacherHubPage.vue:318 现有的手写教师卡）。评分展示收敛为三件套：EmojiRating（表情/单值）、RatingCircle（环形总分）、RatingBar（维度条），并让 CourseDetailPage.vue:121-148 改用 RatingBar 并给它补上 `aria-label` prop，把 `dimensionRatingBarAriaLabel` 的逻辑挪进组件。

</details>

<details><summary>核验记录</summary>

死代码事实核实无误：全仓 grep（含 kebab 形式、含 bots/server/docs）对 TeacherStatsCard/SemesterStatsGrid/RatingDistribution/RatingDisplay/DimensionBars 的真实引用为 0，仅出现在自动生成的 clients/web/src/components.d.ts:33,61,62,72,77 和 clients/web/src/modules/review/__tests__/ratingDisplayPolicy.test.ts:14-17；无 defineAsyncComponent/resolveComponent/<component :is> 动态用法。RatingBar.vue 的唯一消费者确为 DimensionBars.vue:3,15；RatingCircle 确实只在 TeacherProfilePage.vue:29 用一次。三处手写进度条也属实：CourseDetailPage.vue:133 `flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden` + :138 `duration-slow` + :131 `w-16`，SemesterStatsGrid.vue:32-40 `duration-slow ease-smooth`，RatingDistribution.vue:9-14 `duration-slower`，而 RatingBar.vue:3-9 是 `min-w-[60px]` + `duration-700`，样式漂移确实存在。但有三处需要修正：(1) 「常年参与打包」不成立——vite.config.ts:75-77 的 unplugin-vue-components 只在模板实际用到时注入 import，从未被 import 的 SFC 不会进入任何 chunk，components.d.ts 是纯 .d.ts，成本只在 typecheck/lint/认知负担；(2) 「对应 i18n key 永不渲染」不准确——review.stats.noData 在 CourseDetailPage.vue:79 和 :154 是活的，只有 review.stats.insufficient/reviewCount/insufficientHint 随 SemesterStatsGrid 一起死掉；(3) 行数是 269 行（5 个文件），不是约 320 行。方案本身缺一步：ratingDisplayPolicy.test.ts:5-23 在模块顶层 readFileSync 这 4 个文件并在 :38-39 断言其内容，直接删文件会让整个 test 文件 ENOENT 崩掉。综合看这是纯维护性问题，零用户可见影响，P1 偏高。

</details>

---

#### `P2` AboutPage 的 7 条 FAQ 手抄了 7 遍模板，且折叠按钮没有 aria-expanded

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/review/views/AboutPage.vue`

**现状**

AboutPage.vue:44-57 是 FAQ 1 的完整结构（button + ChevronDown + transition + v-show 面板），后面 `@click="toggle(1..6)"` / `v-show="expanded[1..6]"` 在 :87、:116、:148、:177、:209、:250 又原样重复了 6 遍，模板占了文件 372 行中的约 290 行。script 里 `FAQ_COUNT = 7` 已经存在（:332），却只用来初始化数组。所有 7 个 `<button>` 都没有 `aria-expanded`、`aria-controls`，面板也没有 `role="region"` / `id`。

**问题**

改一次 FAQ 的间距或图标要动 7 处，漏改必然视觉漂移；加/删一条 FAQ 要同时改模板、索引和 `FAQ_COUNT`，极易错位（i18n key 是 `faq1q`…`faq7q` 硬编号，删中间一条会全体错位）。无障碍上，屏幕阅读器读到的只是一个普通按钮，听不出当前是展开还是收起。

**建议改法**

把 FAQ 抽成 `const faqs = [{ q: 'review.about.faq1q', a: ['review.about.faq1a1', ...] }, ...]`（或改成 i18n 里的数组结构），模板收敛为单个 `v-for`；给 button 补 `:aria-expanded="expanded[i]"` 和 `:aria-controls="`faq-panel-${i}`"`，面板补对应 `:id` 与 `role="region"`。顺带把这套折叠逻辑（含 onEnter/onLeave 高度动画）提到 `components/ui/Accordion.vue`，因为 components/ui 下目前还没有可复用的折叠组件。

<details><summary>核验记录</summary>

读了 /home/wztxy/Code/StuHelper/clients/web/src/modules/review/views/AboutPage.vue 全文（373 行）。FAQ 1 结构在 :45-80，后面 toggle(1)@:87、toggle(2)@:116、toggle(3)@:148、toggle(4)@:176、toggle(5)@:209、toggle(6)@:250 逐字重复 6 遍（每块都是同一份 class 串 + ChevronDown + transition + v-show），共 :44-273 约 230 行。FAQ_COUNT = 7 在 :332，确实只用于 Array.from 初始化 expanded。全文件 7 个 <button> 只有 type/class/@click，没有任何 aria-* 绑定，面板 div 也没有 id/role（我通读了整份模板，不是 grep 推断）。components/ui 下只有 button/ Button.vue Card.vue dialog/ Empty.vue input/ Loading.vue Pagination.vue SearchBar.vue，确无 Accordion；全仓 grep -ril accordion 只命中 AboutPage.vue 自身（scoped 里的 .accordion-enter-active）。i18n 侧 zh-CN/review.ts:405-426 与 en-US/review.ts:378-399 确为 faq1q…faq7q 硬编号。方案（数组 + v-for + aria-expanded/aria-controls + 抽 Accordion）可行，P2 与「无功能性破坏的可维护性+a11y 债」相符。唯一小瑕疵：模板实际 323 行而非「约 290 行」，不影响结论。

</details>

---

#### `P2` DraftIndicator 的「删除」是一键无确认的破坏性操作，且与旁边的「恢复草稿」语义重叠

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/business/review/DraftIndicator.vue:16`
- `clients/web/src/modules/review/views/PostReviewPage.vue:25`
- `clients/web/src/modules/review/views/PostReviewPage.vue:896`

**现状**

表单每次变更后 700ms 自动存草稿，指示条显示「草稿已保存 刚刚」，右侧常驻 [恢复草稿][删除] 两个小按钮（只要 hasDraft && !saving 就显示）。「删除」直接调 discardCurrentDraft() → draftStore.deleteDraft()，无确认无撤销；「恢复草稿」会为你正在写的这份草稿再弹一次恢复模态（等于把当前内容再覆盖一遍自己）。

**问题**

两个按钮都紧贴着「已保存」状态文案，一个是无意义操作、一个是不可逆破坏操作，而且尺寸小（px-2 py-1，约 24px 高）。用户想确认草稿存没存，手一抖点到「删除」就没了。

**建议改法**

1) 「删除」改为二次确认，直接复用现成的 DraftPromptDialog（PostReviewPage.vue:433-443 已有一个同款用法），避免为此扩展 useToast；确需撤销体验时再单独提 ToastItem 增加 action 的需求。2) 「恢复草稿」按 draftStore.draft（stores/draft.ts:21 已在客户端缓存服务端草稿全文）生成签名，与 currentDraftSignature()（PostReviewPage.vue:778-792）比对，相等时隐藏该按钮——需要抽一个 signatureOf(draft) 复用同一套字段/排序逻辑。3) 顺手修掉配套漏洞：discardCurrentDraft 之后若用户在离开弹窗里选「保留草稿」，应先清空 discardedDraftSignature 再 autosaveDraftNow，否则 :807 的早退让「保留」变成空操作。4) 两个按钮的点击区提到 ≥44px（如 px-3 py-2 或加 min-h）。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

「恢复草稿」只在「服务端草稿 signature ≠ 当前表单 signature」时显示（PostReviewPage 已经有 currentDraftSignature() 可直接比对）；「删除」改为二次确认，或删除后弹一个 5 秒内可「撤销」的 toast（useToast 支持自定义 duration，撤销即重新 saveDraft）。

</details>

<details><summary>核验记录</summary>

事实成立：DraftIndicator.vue:16-31 在 `hasDraft && !saving` 时常驻 [恢复草稿][删除]，两个按钮都是 px-2 py-1 text-xs（约 24px 高，刚踩 WCAG 2.5.8 的 24px 下限）；PostReviewPage.vue:25-33 把 @delete 接到 :896-905 的 discardCurrentDraft，里面直接 await draftStore.deleteDraft()。我按纪律搜过 ElMessageBox.confirm / window.confirm / 自定义确认弹窗：全仓只有 ResourceDetailPage.vue:148 和 ResourceMinePage.vue:223 用 window.confirm，草稿删除路径上确实没有任何二次确认。@restore→:885-894 promptRestoreExistingDraft 会重新拉服务端草稿再弹 DraftPromptDialog，对正在编辑的同一份草稿基本是自我覆盖，语义重叠属实。
但「手一抖就没了」被夸大：删除只清服务端草稿，表单里的标题/正文原样留在页面上；此后任意一次输入都会让 currentDraftSignature() 与 discardedDraftSignature 不等（:821-829），700ms 后自动重新保存。真正的丢失路径是「删除后不再编辑、直接离开」——此时 confirmLeaveWithDraft(:954-963) 先调 autosaveDraftNow，而它在 :807 因签名相等直接 return，用户在离开弹窗里选「保留草稿」也什么都没存下。综合：真实缺陷 + 有数据丢失可能，但需要额外一步用户动作，定 P2 更贴切；且原方案中「5 秒可撤销 toast」这一支不可行——composables/useToast.ts:9-14 的 ToastItem 只有 id/type/message/duration，没有 action/按钮插槽，撤销需要先扩展 toast 组件。

</details>

---

#### `P2` 「评价数」有三套渲染，其中 CourseCard 是一个没有单位的裸数字

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/business/review/CourseCard.vue`
- `clients/web/src/components/business/review/CourseListItem.vue`
- `clients/web/src/modules/course/views/CourseListPage.vue`

**现状**

同一个 course.reviewCount 有三种呈现：CourseCard.vue:13-15 直接 {{ course.reviewCount }}，无单位、无 i18n，紧挨着一个院系名的 pill；CourseListItem.vue:10 用 t('review.course.reviewCountBadge', { count })；CourseListPage.vue:343-345 用 t('review.courseList.reviewCount', { count })（zh 为「{count} 条评价」）。两个组件的骨架也高度重合：都是 router-link → /courses/:id、都渲染课程名 + 评价数 + 一项次要信息，只是 CourseCard 是 flex + glass-card（用在 MyFavoritesTab.vue:25），CourseListItem 是 grid + 高亮当前课程（用在 DepartmentSidebar.vue:67）。

**问题**

在「我的收藏」里，用户看到的是「数据结构   计算机学院  12」——12 是学分、评价数还是排名，完全靠猜，这是三处里唯一没有标签的一处，而且是硬编码数字、不走 i18n（英文环境下也一样是个裸数字）。两个组件维护同一种信息却各写各的，任何一次「评价数改成显示平均分」的改动都要改两遍且容易只改一半。

**建议改法**

先把 CourseCard.vue:13-15 改成 t('review.courseList.reviewCount', { count: course.reviewCount })，与另外两处对齐（这一步 5 分钟）。再合并成一个 CourseRow.vue，props 为 { course, variant: 'card' | 'compact', active? }，variant 只控制外层 class（glass-card/flex vs grid/hover），信息结构与文案共用一份；MyFavoritesTab 和 DepartmentSidebar 各传各的 variant。

<details><summary>核验记录</summary>

核实无误：CourseCard.vue:13-15 就是 `{{ course.reviewCount }}`，无单位无 i18n，且紧邻 :10-12 的院系名 pill；CourseListItem.vue:10 用 t('review.course.reviewCountBadge')（zh-CN/review.ts:33 = '{count}评'）；CourseListPage.vue:343-345 用 t('review.courseList.reviewCount')（zh-CN/review.ts:302 = '{count} 条评价'）。两组件骨架确实高度重合（都是 router-link → /courses/:id + 课程名 + 评价数 + 一项次要信息），CourseCard 是 flex+glass-card 且只被 MyFavoritesTab.vue:25 使用，CourseListItem 是 grid+isActive 高亮且只被 DepartmentSidebar.vue:67 使用。第一步改 t('review.courseList.reviewCount') 零风险。P2 合理。

</details>

---

#### `P2` 举报理由点一下立即上报，没有二次确认也没有关闭入口

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/business/review/ReviewCard.vue:246`
- `clients/web/src/components/business/review/useReviewReport.ts:25`

**现状**

点旗帜图标展开一个内联面板，5 个理由 chip（px-3 py-1.5，约 28px 高，flex-wrap 排列，间距 8px），点任意一个立刻 POST 举报并 toast 成功；面板没有关闭按钮，点外面不会收起，Esc 也无效，只能再点一次旗帜图标；reporting 期间 chip 仅 disabled，没有 loading 文案。同一个卡片里的删除操作反而做了规范的内联二次确认（第 212-243 行）。

**问题**

举报是针对他人内容的不可撤销动作，却是全卡片里最容易误触的：28px 高的小标签、间距 8px、单击即提交。用户展开面板后想反悔，界面上没有任何「取消」。

**建议改法**

复用同卡片已有的内联确认样式：点 chip 后先切到「确认举报：{理由}」+[取消][确认] 状态再提交；面板补一个「取消」按钮并支持 Esc / 点击外部关闭；提交中把被点的 chip 文案换成 loading 态。

<details><summary>核验记录</summary>

ReviewCard.vue:246-263 的举报面板与描述一致：v-if="showReportMenu" 的内联 div 上没有任何 @keydown.esc、没有 click-outside 指令/监听，面板内只有一行标题 <p>（:250）和 v-for 出来的 5 个 chip（:252-261，class 为 'text-xs px-3 py-1.5 rounded-full ...'，text-xs 行高 16px + py-1.5 上下各 6px ≈ 28px 高，容器 flex flex-wrap gap-2 = 8px 间距），chip 仅 :disabled="reporting"，@click="handleReport(reason)" 直接提交，无「取消」按钮；关闭只能靠 :173-182 的旗帜按钮再点一次 toggleReportMenu（useReviewReport.ts:21-23 只是取反）。useReviewReport.ts:25-36 确认 handleReport 立即 POST api.review.reportReview 并 toast.success，无任何 ElMessageBox.confirm / window.confirm / 自定义确认弹窗（全仓 grep 无 ElMessageBox 使用）。对照组也属实：同一卡片的删除在 ReviewCard.vue:212-243 有规范的内联二次确认（confirmingDelete + 取消/确认按钮 + @keydown.esc + :509-518 的焦点管理），复用其样式与交互模式完全可行；已有 e2e tests/e2e/review-actions.spec.ts:512-535 断言点击理由后 POST body 为 { reason: 'spam' }，改造时需同步在该用例里补一次「确认」点击，属预期内的测试调整，不影响方案成立。

</details>

---

#### `P2` 可直接删除：TeacherFilter.vue 无任何引用、ReviewPage 的 router-view 与 watch 永不生效、home 语言包 6 组未用 key

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/business/review/TeacherFilter.vue`
- `clients/web/src/modules/review/views/ReviewPage.vue`
- `clients/web/src/i18n/locales/zh-CN/home.ts`
- `clients/web/src/i18n/locales/en-US/home.ts`

**现状**

① TeacherFilter.vue（42 行，含 review.filter.teacher / review.filter.all 两个 i18n key）全仓库搜 TeacherFilter 和 teacher-filter 只命中它自己和自动生成的 components.d.ts:76，零使用。② router/index.ts 里 routes 数组（:90 起）没有任何 children，所有路由都是扁平顶层记录，所以 route.matched.length 恒为 1，ReviewPage.vue:73-75 的 hasChildRoute 恒为 false —— :52 的 <router-view v-if="hasChildRoute" /> 永远不渲染；同理 :78-80 那个「路由切到具体课程后自动关闭侧边抽屉」的 watch，因为 /courses/:id 是平级路由、ReviewPage 会整体卸载，也永远不会触发。③ zh-CN/home.ts:5-22 的 title/subtitle/postReview/features.{review,resource,spoc} 和 :45-54 的 landing.features.{toolbox,community} 全部无引用（唯一还在用的顶层 key 是 home.explore，见 HeroSection.vue:39），en-US/home.ts 同样位置同样未用。

**问题**

死代码给后来者制造假象：看到 TeacherFilter 会以为教师筛选已经做好了；看到 ReviewPage 的 router-view 会以为课程详情是嵌套渲染的，从而按错误的模型去改布局；toolbox / community 两组文案会让人以为这两个模块已经排期。三处都是纯负担，没有任何行为依赖它们。

**建议改法**

删除 components/business/review/TeacherFilter.vue；i18n 只删 `review.filter.teacher`（zh-CN/review.ts:270 及 en-US 对应行），**保留 `review.filter.all`**，它被 CourseDetailPage.vue:578 的教师筛选 chips 使用；删除 ReviewPage.vue:52 的 router-view（改为直接渲染 `<ReviewFeed />`）、:73-75 的 hasChildRoute、:78-80 的 watch，并同步移除 script 里已不再使用的 computed/watch/useRoute 导入（:61-62，注意 route 若无其他使用者也一并删）；两个 home 语言包按原文删除未用 key。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

删除 components/business/review/TeacherFilter.vue 及其 review.filter.teacher / review.filter.all 两个 key（确认 review.filter.* 没有其他使用者后）；删除 ReviewPage.vue:52 的 router-view、:73-75 的 hasChildRoute 和 :78-80 的 watch（连同 computed/watch 的 import）；删除两个 home 语言包里上述未用 key。若确实还要做教师筛选，也该在 CourseDetailPage 接上再恢复，而不是留一个孤儿组件。

</details>

<details><summary>核验记录</summary>

三项事实均核实为真。① 全仓 grep `TeacherFilter|teacher-filter` 只命中 components.d.ts:76（自动注册，不算引用）与 tests/e2e/course-browse.spec.ts:660/678/685 的 review fixture id（`teacher-filter-wang` 等，是测评数据 id，与组件无关），组件零使用。② router/index.ts 全文无 `children`、全仓无 `addRoute`，routes 从 :90 起全是扁平顶层记录，故 route.matched.length 恒为 1，ReviewPage.vue:73-75 的 hasChildRoute 恒 false，:52 的 `<router-view v-if="hasChildRoute" />` 永不渲染；ReviewPage 只挂在 /courses/reviews（router/index.ts:323-326，name "review"，无 :id 参数），/courses/:id 是平级路由，:78-80 的 watch 确为死码。③ home 语言包：仅 home.explore（HeroSection.vue:39）与 home.landing.{hero,features.reviewCenter/teacherProfile/resourceHub,stats,footer}（HomePage.vue:92-188）在用，zh-CN/home.ts:5-6/8/9-22 与 :45-54、en-US/home.ts 同位置确实无引用，且无模板字符串动态拼 home.* 的用法。严重度 P2 合理。但原方案有一处会破坏现有功能：`review.filter.all` 并非 TeacherFilter 独占，CourseDetailPage.vue:578 `{ teacherID: null, teacherName: t('review.filter.all') }` 正在使用（zh-CN/review.ts:269-272 的 filter 块）；按原文照删会让课程详情页教师筛选 chip 显示成裸 key。

</details>

---

#### `P2` 同一条评测存在两套完全不同的卡片实现：信息流用 ReviewCard，课程详情页手写了另一套（图标、颜色、功能都不一样）

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/web/src/components/business/review/ReviewCard.vue:135`
- `clients/web/src/modules/review/views/CourseDetailPage.vue:214`
- `clients/web/src/modules/review/useReviewVoting.ts:17`
- `clients/web/src/modules/review/useReviewReplies.ts:22`

**现状**

ReviewCard：点赞用实心 Heart、激活色 primary（蓝紫）、按钮无 title 提示、不展示学期/成绩/争议提示，标题行还恒定预留 `pr-20` 给管理员工具栏（第 45 行，普通用户也白白少 80px 标题宽度）。CourseDetailPage 第 214-374 行手写了 160 行几乎同功能的模板：点赞用 ThumbsUp、激活色是专门的 vote-up 绿 / vote-down 红 token、有 title 提示、有学期、有成绩标签、有 ControversialBadge，但没有编辑/删除/举报。投票与回复也各有一套 composable（useReviewVote/useReviewReply vs useReviewVoting/useReviewReplies）。

**问题**

用户在信息流点了赞是蓝紫爱心，进到课程页同一条评测变成绿色大拇指；发布成功后落地的正是课程详情页那套，那里既没有编辑删除、点赞图标也和刚才看到的不一样。--color-vote-up/down 这组专门的 token 只有一个页面在用，等于设计规范失效。

**建议改法**

拆成两步，先做低风险的收敛：(1) ReviewCard 侧对齐设计规范——点赞换 ThumbsUp、激活色改用 --color-vote-up/--color-vote-up-active 与 --color-vote-down（tailwind.css:73-76），:135-182 的按钮补 :title，补 termName / grade / ControversialBadge 三个展示位，:45 的 `pr-20` 改成 `:class="canManageReviews ? 'pr-20' : ''"`（照抄 CourseDetailPage.vue:252）。这一步不动 CourseDetailPage，零回归风险。(2) 再评估把 CourseDetailPage.vue:213-374 换成 <ReviewCard>，前置条件是：给 ReviewCard 的点赞/点踩加上 `review-like-${id}` / `review-dislike-${id}` testid 以保住四个 e2e 用例；标题在课程详情页上下文里改为不可点或指向评测锚点；先明确哪套 composable 是正统（建议保留带未登录跳 /login 的 useReviewVoting 语义，把它下沉进 ReviewCard），迁移完成并跑通 e2e 后才删除并行实现——在此之前不要先删 useReviewVoting/useReviewReplies。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 CourseDetailPage 的评测项替换成 <ReviewCard>，给 ReviewCard 补 termName / grade / ControversialBadge 三个展示位和 `:title` 提示，`pr-20` 改成 `:class="canManageReviews ? 'pr-20' : ''"`；删掉 CourseDetailPage 里那 160 行模板和 useReviewVoting/useReviewReplies 这套并行实现；点赞统一为 ThumbsUp + vote-up 色。

</details>

<details><summary>核验记录</summary>

事实全部核对属实：ReviewCard.vue:135-147 点赞用 Heart + `!text-primary bg-primary/[0.08]`，:135-209 全部按钮只有 :aria-label 没有 :title，整份文件（我通读了 1-548 行）不含 termName / grade / ControversialBadge，:45 的标题行是硬编码 `pr-20`；CourseDetailPage.vue:213-374 确为手写的 ~160 行同功能模板，:264 有学期、:309-313 有成绩标签、:316 有 ControversialBadge、:337-362 用 ThumbsUp/ThumbsDown + vote-up/vote-down token 且带 :title，但没有举报/本人编辑/删除（:376-411 之后直接进回复区）。--color-vote-up/-active/-down/-active 定义在 styles/tailwind.css:73-76（暗色 277-280、344-347），全仓引用只有 CourseDetailPage.vue:340/341/353/354，确实只有一个页面在用。两套 composable 并存也属实（useReviewVote.ts / useReviewReply.ts vs modules/review/useReviewVoting.ts:17 / useReviewReplies.ts:22）。router/index.ts:339-345 course-reviews → CourseDetailPage，PostReviewPage.vue:1060 发布成功后正是跳这里，落地页无编辑删除属实；ReviewCard.vue:45 的固定 pr-20 让普通用户白白损失 80px 也属实（CourseDetailPage.vue:252 已经写成 `canManageReviews ? 'pr-20' : ''`，证明是 ReviewCard 漏改）。
下调理由：两个页面各自功能完整、无报错、无 a11y 违规，属于一致性/重复实现问题，用户可见影响是图标与配色不统一 + 落地页缺本人编辑删除，量级低于同批其它条目，P1 偏高。原方案「删掉那 160 行模板和 useReviewVoting/useReviewReplies」一刀切也有风险：clients/web/tests/e2e 下 journey-review.spec.ts:255/261、course-browse.spec.ts:1497、review-flow.spec.ts:668、review-actions.spec.ts:391 都依赖 CourseDetailPage 的 `review-like-*`/`review-dislike-*` testid，而 ReviewCard 没有这些钩子；ReviewCard.vue:46-51 的标题是指向 /courses/{id}/reviews 的 router-link，直接搬到课程详情页会变成自引用链接；两套 composable 的鉴权语义也不同（useReviewVoting.ts:43-53 未登录跳 /login，ReviewCard 的 useReviewVote 完全不查登录、靠 :432 showActions 兜底）。

</details>

---

#### `P2` 搜索结果里的课程是全站第三套课程卡，且页内两组卡片视觉与落点都不一致

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/review/views/SearchPage.vue`
- `clients/web/src/components/business/review/CourseCard.vue`
- `clients/web/src/components/business/review/CourseListItem.vue`

**现状**

SearchPage.vue:249-268「有测评的课程」= `bg-bg-card rounded-xl shadow-card p-4` + hover 描边 + `stagger-item` 入场动画，跳 `/courses/:id/reviews`；:282-295「无测评的课程」= 同结构但去掉了 `shadow-card`、去掉 `hover:shadow-md`、去掉入场动画，跳 `/courses/:id`。另外两套是 CourseCard.vue（`glass-card hover-lift` 毛玻璃，用于收藏页）和 CourseListItem.vue（两列 grid 紧凑行，用于 DepartmentSidebar）。

**问题**

同一个「一门课」的实体在收藏页、侧栏、搜索结果里长成三个样子，学生认不出这是同一种东西。更糟的是搜索结果内部：两组卡片看起来几乎一样，点下去却一个进测评列表、一个进课程详情，用户无法预判会跳到哪；而且「无测评」那组少了阴影和动画，看起来像被禁用/加载失败，而不是「这门课还没人评」。

**建议改法**

1) 卡片统一：抽 CourseResultCard（或给 CourseCard 加 variant/to prop），把 SearchPage.vue:248-266 与 :281-294 两组、以及 MyFavoritesTab 的 CourseCard 收敛到同一排版与阴影；两组差异只用 badge 表达（有测评显示「N 条测评」，无测评显示 outline 风格的「暂无测评 · 去抢首评」），并给「无测评」组补上 stagger-item，避免它看起来像禁用态。CourseListItem 是侧栏导航行、信息密度诉求不同，可保留为独立组件，不必强并。2) URL 收敛的方向应反过来：以引用更多的 `/courses/:id/reviews`（name: course-reviews）为规范形态，把 SearchPage.vue:285 改成 /reviews；若确要只保留一条路由，则删 course-reviews 并把 router/index.ts 改为 `/courses/:id/reviews` redirect 到 course-detail，同时必须改 AppHeader.vue:276 的 `route.name !== "course-detail" && route.name !== "course-reviews"` 判断与 AppHeader.test.ts:342、PostReviewPage.vue:1060 的 name 跳转，否则写测评按钮的课程上下文会失效。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

抽一个 `CourseResultCard`（或直接扩展 CourseCard 加 `variant`/`href` prop）统一这三处的排版与阴影；搜索结果两组统一跳 `/courses/:id`（详情页本身已有测评区），差异只用 badge 表达 —— 有测评显示「N 条测评」，无测评显示一个 outline 风格的「暂无测评 · 去抢首评」badge，把空课变成投稿入口而不是视觉降级。

</details>

<details><summary>核验记录</summary>

视觉部分属实：SearchPage.vue:248-266「有测评」卡为 `bg-bg-card rounded-xl shadow-card p-4 ... border border-transparent hover:border-primary/50 hover:shadow-md transition-all stagger-item` + :254 animationDelay，:281-294「无测评」卡为 `bg-bg-card rounded-xl p-4 ... border border-transparent hover:border-primary/50 transition-all`，确实少了 shadow-card（tailwind.css:89 有定义）、hover:shadow-md 与 stagger-item（tailwind.css:607，含 opacity:0 + fade-in-up）；CourseCard.vue:4 是 `glass-card hover-lift` 且只被 MyFavoritesTab.vue:25 使用，CourseListItem.vue:4 是 `grid grid-cols-[minmax(0,1fr)_auto]` 紧凑行且只被 DepartmentSidebar.vue:67 使用，「三套课程卡」成立。但本条最重的那句「点下去却一个进测评列表、一个进课程详情，用户无法预判会跳到哪」是错的：router/index.ts:331-345 里 `/courses/:id`(course-detail) 与 `/courses/:id/reviews`(course-reviews) 挂的是同一个 `@/modules/review/views/CourseDetailPage.vue`，且该组件全文不读 route.name/route.path 做分支（只有 :614 与 :845 取 path/fullPath 用于埋点与回跳），两组卡片落到的是像素级相同的页面，唯一差异是 meta.titleKey（routes.courseDetail「课程详情」vs routes.courseReviews「课程测评」）导致的标签页标题。所以「落点不一致」只是 URL 不一致，不存在用户预期落差，问题降级为纯视觉一致性 + 路由冗余，P1 偏高。另外方案里「统一跳 /courses/:id」方向选反了：全站 4 处链接（ReviewCard.vue:47、CourseListPage.vue:336、SearchPage.vue:252、PostReviewPage.vue:1060 的 name:'course-reviews'）用的是 /reviews 形态，只有 CourseCard/CourseListItem/TeacherProfilePage/CommandPalette 用裸 /courses/:id。

</details>

---

#### `P2` 教师主页课程列表没有空态；同时全站教师名都不可点进教师主页

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/review/views/TeacherProfilePage.vue`
- `clients/web/src/modules/review/views/CourseDetailPage.vue`

**现状**

TeacherProfilePage.vue:111-127 课程列表是一个裸 `v-for`，没有 `v-else` 空态；同页评分趋势（:103-105）有 `t('common.empty.data')` 兜底、最新测评（:171-176）有 `t('teaching.profile.reviewsEmpty')` 兜底。入口方面：`/teachers/:id` 的全部来源只有 TeacherHubPage.vue:318 和 CommandPalette.vue:333，而 CourseDetailPage.vue:196-205 的教师 chip 是筛选按钮（`@click="selectTeacher"`），不是链接。

**问题**

新入库或课程尚未关联的老师打开主页，「任课课程」标题下面是一片纯空白 —— 同一屏里另外两个区块都有明确空态，唯独这里像渲染挂了。导航上更严重：学生在课程详情页看到「张三」，或在测评卡上看到教师名，没有任何途径点进这位老师的主页，整个教师主页功能只能靠教师 Hub 列表页触达，白做。

**建议改法**

保持原方案的两步，只调整优先级与措辞：1) TeacherProfilePage.vue:111 补 `v-if="teacher.courses.length"`，并加 v-else 空态，样式直接复用同页 :103-105 / :173-175 那个 `rounded-lg border border-border-light bg-bg-elevated/40 p-8 text-center text-sm text-text-muted` 块，文案新增 teaching.profile.coursesEmpty（zh-CN + en-US 同步）；:116 的 hover:pl-6 换成 hover:translate-x-1（配合已有的 transition-all duration-fast），消除 hover 时 padding 变化引起的行内重排。2) 教师入口按投入产出排序：先做 ReviewCard.vue:67 —— 在 review.teacherID 存在时把教师名包成 router-link 指向 /teachers/{teacherID}，teacherID 为空则保持现有 span；CourseDetailPage.vue:188-208 的 chip 维持筛选语义不变，只在 selectedTeacherID 非空时于测评区上方加一条「查看 {name} 的全部评价 →」链接。这两步都是纯增量，不影响现有筛选行为。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

1) TeacherProfilePage.vue:111 补 `v-if="teacher.courses.length"` + `v-else` 空态，文案复用 `common.empty.data` 或新增 `teaching.profile.coursesEmpty`；顺带把 :116 的 `hover:pl-6` 换成 `hover:translate-x-1`，避免 hover 时 padding 变化引起整行文字重排抖动。2) CourseDetailPage.vue:196 的 chip 保持筛选语义，但在 chip 选中后于测评区上方加一条「查看 张三 的全部评价 →」链接指向 `/teachers/:id`；ReviewCard 上的教师名同样加链接。

</details>

<details><summary>核验记录</summary>

事实全部核实无误。TeacherProfilePage.vue:111-127 确为裸列表：:111 容器 div、:112-113 `<router-link v-for="course in teacher.courses">`、:116 class 含 `hover:pl-6`，整段没有 v-else 空态；同页 :103-105 有 `t('common.empty.data')` 兜底、:173-175 有 `t('teaching.profile.reviewsEmpty')` 兜底，:145-165 还有骨架屏与错误重试，唯独课程列表裸奔，属实。teacher.courses 由 :380-395 的 readTeacherPayload 强制校验为数组（Array.isArray 检查 + map），所以 `v-if="teacher.courses.length"` 安全可写。入口方面 grep 全仓 `/teachers/${` 只有 TeacherHubPage.vue:318、CommandPalette.vue:333，外加已死的 TeacherStatsCard.vue:42（第 1 条已证其零引用），CourseDetailPage.vue:188-208 的教师 chip 确是 `<button type="button" ... @click="selectTeacher(teacher)">` 筛选按钮，ReviewCard.vue:67 的教师名确是裸 `<span class="text-teacher-tag font-medium">`。方案可行性也成立：chip 有 teacher.teacherID，Review schema（api.gen.ts:3787-3788）同时有 teacherID?: number|null 与 teacherName，ReviewCard 加链接时对 teacherID 为空的做 v-if 降级即可。需要修正的只有严重度：「整个教师主页功能只能靠教师 Hub 列表页触达，白做」夸大了——/teachers 是主导航一级入口（AppHeader.vue:264 nav.teacher、FloatingModuleNav.vue:105），再加全局命令面板，教师页并非不可达；缺的是上下文入口与一个空态兜底，无流程阻塞、无数据错误，属于打磨项。

</details>

---

#### `P2` 表情评分控件没有选中状态语义，读屏听不出打了几分，错误提示也不播报

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/components/business/review/EmojiRatingInput.vue:3`
- `clients/web/src/components/business/review/EmojiRatingInput.vue:23`
- `clients/web/src/components/business/review/EmojiRatingInput.vue:55`
- `clients/web/src/modules/review/views/PostReviewPage.vue:257`

**现状**

5 个 button 的可访问名是 `${label} ${level}`（如「教学质量 3」），没有 aria-pressed，也没有 role=radiogroup/aria-checked；选中态只靠 `scale-[1.2]` + inline color + box-shadow 表达。错误文案是紧跟其后的普通 <span>，没有 role="alert"，也没有通过 aria-describedby 关联到任何控件。维度名是父级 PostReviewPage 里的一个裸 <span>，靠 props.label 二次拼接。

**问题**

读屏用户逐个 tab 过去只会听到「教学页量 1、2、3、4、5 按钮」，无法知道当前选中的是哪一档，也听不到「请为每个维度打分」；色觉障碍用户只能靠 1.2 倍缩放辨认选中项。评分是必填项，这直接导致这类用户无法完成发布。

**建议改法**

保留原生 <button>（与本控件可反选的行为一致），逐个加 :aria-pressed="modelValue === level"；分组语义改用 role="group"，在 PostReviewPage.vue:254 的维度名 <span> 上加 :id="`rating-label-${dim.key}`"，由 EmojiRatingInput 新增 labelledBy prop 绑到外层 div 的 aria-labelledby（保留现有 fieldset/legend 不变）；错误 <span> 加 role="alert" 与 :id，并挂到外层 group 的 :aria-describedby；选中态在颜色之外补一个非颜色标记（现有 border-2 在选中时改实线加粗，或在图标下加一个小圆点），未选中的 opacity-40 与选中的 1.2 倍缩放继续保留。若确实要用 radiogroup，必须同时实现 roving tabindex + 左右/上下方向键切换，并放弃点击清零的交互。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

外层 div 加 `role="radiogroup"` + `aria-labelledby` 指向维度名 span 的 id，按钮改 `role="radio" :aria-checked="modelValue === level"`；选中态在颜色之外加一个非颜色标记（现有 border-2 可以在选中时加粗/加实心底点）；错误 span 加 `role="alert"` 并把 id 挂到 radiogroup 的 aria-describedby 上。

</details>

<details><summary>核验记录</summary>

事实全部属实：EmojiRatingInput.vue:3-22 五个 button 只有 :aria-label/:title=buttonLabel(level)（:55-57 返回 `${label} ${level}`），无 aria-pressed / role=radio / aria-checked；选中态仅由 :11-16 的 'scale-[1.2] selected' + 内联 color/borderColor + :61-64 的 box-shadow 表达；错误文案 :23-28 是裸 <span>，无 role="alert"、未被任何 aria-describedby 引用（对比 PostReviewPage.vue:290-302 标题输入框确实做了 aria-describedby 关联）；维度名确为 PostReviewPage.vue:254-256 的裸 <span>，无 id，EmojiRatingInput 在 :257 处接收 :label=dim.name 二次拼接。降级/改方案的理由有二：(1) 严重度：外层已有 <fieldset>+<legend>（PostReviewPage.vue:223-229），控件可聚焦、可激活、提交不被 aria 缺失阻断，读屏用户仍能打分与提交，只是无法确认当前选中档位与听不到 review.post.ratingMissing('请为每个维度打分')，属严重可用性缺陷但非完全阻断，P2 更贴切；(2) 原方案的 role=radiogroup/role=radio 与本控件的实际交互相冲突——toggle() 在 :51-53 允许再次点击已选项清零（ratingTip 明确宣传『点击已选中的图标可取消选择』，zh-CN/review.ts:337），radio 语义无法表达『回到未选中』，且 ARIA radiogroup 需要 roving tabindex + 方向键漫游，原方案未涉及，照做会得到 5 个都在 tab 序里、方向键失效的半成品 radiogroup。

</details>

---

#### `P2` 评测正文整块被包成 role=button 且 aria-label 覆盖了内容，读屏读不到评测正文，鼠标选中文字会误触展开

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/components/business/review/ReviewCard.vue:103`

**现状**

正文是一个 `role="button" tabindex="0" :aria-label="t('review.review.expandContent')" :aria-expanded` 的 div，v-text 输出评测内容，click/enter/space 都切换展开。`shouldTruncate` 只有在内容 >200 字时才加 line-clamp-3，但点击行为对短评测同样生效。

**问题**

role=button 上的 aria-label 会覆盖内部文本作为可访问名，读屏在这个元素上只会读出「展开评价内容 按钮」，评测正文本身读不出来——而正文正是这个页面唯一有价值的信息。鼠标用户想拖选一段正文引用给同学，一松手卡片就展开/收起了。

**建议改法**

正文改回普通 div（保留 v-text 防 XSS），只在 shouldTruncate 为真时在正文下方渲染一个真正的「展开全文 / 收起」按钮（带 aria-expanded 指向正文 id）；展开回复继续交给已有的评论按钮，两个动作解耦。

<details><summary>核验记录</summary>

ReviewCard.vue:103-114 与描述逐字吻合：正文 div 带 role="button" tabindex="0" :aria-label="t('review.review.expandContent')"（zh-CN/review.ts:49='展开全文'、en-US:64='Expand content'）、:aria-expanded="isExpanded"，click/keydown.enter/keydown.space 均调用 toggleExpand（:538-543），内容用 v-text 输出。line-clamp-3 只在 `!isExpanded && shouldTruncate` 时生效（:105），而 shouldTruncate 定义在 :439 为 content.length > 200，点击行为对短评测同样生效，符合描述。可访问名断言正确：aria-label 在 accname 计算中优先于内容，且 role=button 是 children-presentational，读屏在该元素上只报「展开全文 按钮」，正文本身不作为可访问名暴露。鼠标拖选后 mouseup 落在同一元素会触发 click，确实会误触展开/收起。方案可行且不破坏现有功能：正文保留 v-text；展开全文按钮需与回复展开解耦（当前 isExpanded 同时控制正文截断与 :309 的回复区，且 :161-170 的评论按钮也调用 toggleExpand），拆成 contentExpanded / repliesExpanded 两个状态即可，现有 ReviewCard.locked.test.ts / ReviewCard.delete.test.ts 均未断言 expandContent 或该 div 的 role。

</details>

---

#### `P2` 评课域里有三个零引用组件、两个零引用 composable，投票的乐观更新还被手写了两遍，提示系统也是两套

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/composables/useOptimisticUpdate.ts:17`
- `clients/web/src/composables/useAsyncData.ts:18`
- `clients/web/src/components/business/review/ContentQualityTip.vue:1`
- `clients/web/src/components/business/review/RatingDisplay.vue:1`
- `clients/web/src/components/business/review/DimensionBars.vue:1`
- `clients/web/src/modules/auth/views/LoginPage.vue:127`

**现状**

useOptimisticUpdate / useAsyncData 在业务代码里 0 引用（只有各自的单测在 import）；而投票的「乐观更新 + 失败回滚 + toast」在 useReviewVote.ts:52-89 和 useReviewVoting.ts:43-80 各手写了一遍，两者行为还不一致（前者失败时抖动卡片，后者没有；前者未登录不可达，后者会跳登录）。ContentQualityTip.vue / RatingDisplay.vue / DimensionBars.vue 在所有 .vue 里 0 引用，只出现在自动生成的 components.d.ts 和一个读源码字符串的策略测试里。提示系统方面，全局挂载的是 Toast.vue + useToast，但 LoginPage 还在用 element-plus 的 ElMessage（main.ts 为此单独引了 message 的 css）。

**问题**

改投票逻辑要改两个文件，很容易只改一处造成两个页面行为不一致（现在已经不一致了）；三个没人用的展示组件让人误以为有现成方案；同一个产品出现两种视觉完全不同的错误提示（右上角自绘 Toast vs Element Plus 顶部消息）。

**建议改法**

1) 删掉 ContentQualityTip（或把它接到发布页的敏感词警告里，见第 5 条）、RatingDisplay、DimensionBars，同步删掉 ratingDisplayPolicy.test.ts 里对 RatingDisplay 的断言；2) 让两个投票 composable 都基于 useOptimisticUpdate 实现，或删掉 useOptimisticUpdate/useAsyncData；3) LoginPage 的 2 处 ElMessage 换成 useToast，之后 main.ts 里 element-plus 的 message 样式引入也可一并去掉。

<details><summary>核验记录</summary>

四组事实逐条核对通过。(1) 全仓 grep（排除 node_modules）显示 useOptimisticUpdate / useAsyncData 仅出现在自身实现与 src/composables/__tests__/ 下的同名单测中，业务代码零引用。(2) ContentQualityTip.vue / DimensionBars.vue / RatingDisplay.vue 只出现在自动生成的 src/components.d.ts:21/33/61，以及 src/modules/review/__tests__/ratingDisplayPolicy.test.ts:14+38（该测试用 readFileSync 读源码字符串，断言 ratingDisplay 含 'getRatingFacePath'），kebab-case 用法（content-quality-tip / rating-display / dimension-bars）在 .vue 中同样零命中，确认三者未被任何模板使用；原方案已明确要同步删除该测试断言，可行。(3) 投票编排确实写了两遍：components/business/review/useReviewVote.ts:52-89 与 modules/review/useReviewVoting.ts:43-80 行号完全对得上，两者都自行完成『写入乐观状态 → 调 api.review.voteReview → catch 里回滚 → toast.error(voteFailed)』；行为差异属实——前者在失败时置 shaking + 抖动定时器（:82-83），后者没有；后者在 :44-53 做 bootstrapSession/未登录跳 login，前者没有（ReviewCard 只在 showActions=isAuthenticated && !isHidden 时渲染投票按钮，:130-159，故未登录不可达，描述准确）。唯一需要注意的措辞收敛：两者的状态数学复用了 @stuhelper/shared/review 的 createReviewVoteState/applyOptimisticVote/getDisplayVoteCount，重复的是编排与副作用层而非算法，但『改一处漏一处』的维护风险与已存在的不一致成立。(4) 提示双轨属实：LoginPage.vue:68 import { ElMessage }，:127 与 :138 两处 ElMessage.error；main.ts:2 单独引入 'element-plus/es/components/message/style/css'，全仓再无其他 ElMessage 使用（App.vue 只用 ElConfigProvider，其样式由 main.ts:3 单独引入，不受影响），故替换成 useToast 后删除该 css 引入可行。

</details>

---

### 首页与课程发现

共 4 条：P0 0 / P1 3 / P2 1

#### `P1` /courses 热门课程区没有骨架屏、空态和错误重试，失败只有 4 秒后自动消失的 snackbar

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/course/views/TeachingHubPage.vue`
- `clients/web/src/modules/course/views/TeacherHubPage.vue`

**现状**

TeachingHubPage.vue:176 整个热门课程 section 的渲染条件就是 v-if="hotCourses.length > 0"。:520-525 并发四个请求，:558-568 里 getHotCourses 失败就把 hotCourses 置空并调 showError()。showError（:361-367）写的是同一个 errorMessage，:364-366 设置 4 秒定时器自动清空，渲染成 :203-212 的底部 snackbar。对比 TeacherHubPage.vue:290-297 有 shimmer 骨架、:299-310 有错误文案 + 重试按钮、:370-373 有空态；CourseListPage.vue:272-277 也有 9 个 SkeletonCard。

**问题**

① 加载期间该区块根本不存在于 DOM，数据回来时页面突然多出一整块，出现明显跳动；② 请求失败时区块永久隐形，用户只会以为「这个平台没有热门课程」，没有任何重试入口；③ 四个请求全挂时 showError 被连调 4 次覆盖同一个 errorMessage，用户只看到一条泛化的「加载失败」，4 秒后消失，无从知道哪块坏了；④ 同一产品的两个 Hub 页对同类失败给出完全不同的处理方式，一致性也断了。

**建议改法**

给热门课程加 hotLoading / hotError 两个状态，照抄 TeacherHubPage 的三段式：loading 时渲染 6 个和卡片等高的 shimmer 占位；hotError 时在 section 内渲染「加载失败 + 重试」按钮（重试只重发 getHotCourses）；成功但为空时渲染「还没有热门课程，去写下第一条评价」并链到 { name: 'course-review-post' }。snackbar 保留给真正的全局失败，不要当作分区错误的唯一反馈。

<details><summary>核验记录</summary>

逐行核实无误：TeachingHubPage.vue:176 整个 section 的条件就是 v-if="hotCourses.length > 0"（无 loading/error 分支）；:520-525 四个 Promise.allSettled 请求；:558-568 hot 失败置空 + showError；showError（:361-367）复用同一个 errorMessage 且 4000ms 自动清空，四个失败分支（538/543/551/555/563/567/575/579）确实会互相覆盖成一条泛化「加载失败」；渲染点是 :203-212 的底部 snackbar。对照组也属实：TeacherHubPage.vue:290-297 shimmer 骨架、:299-310 错误文案+重试按钮、:370-373 空态；CourseListPage.vue:272-277 九个 SkeletonCard。方案（hotLoading/hotError 三段式 + 只重发 getHotCourses）可行，{ name: 'course-review-post' } 路由存在（router/index.ts:347-357）。P1 合理。

</details>

---

#### `P1` 课程发现被拆成三个页面三个搜索框，且详情页有两个 URL

> 核验确认　|　工作量：L

**位置**

- `clients/web/src/modules/course/views/TeachingHubPage.vue`
- `clients/web/src/modules/course/views/CourseListPage.vue`
- `clients/web/src/modules/review/views/ReviewPage.vue`
- `clients/web/src/router/index.ts`
- `clients/web/src/modules/home/views/HomePage.vue`

**现状**

三个入口：/courses（TeachingHubPage，搜索框带自动补全下拉，query 只存在 usePinyinSearch 的本地 ref 里，见 :269，不写 URL）、/courses/list（CourseListPage，搜索框 + 院系折叠，:150 用 router.replace 把 q 写进 URL）、/courses/reviews（ReviewPage，没有搜索框，只有院系侧栏），另有 /search 高级搜索。TeacherHubPage.vue:121 也把 q 写 URL。首页两个相邻控件分别去往不同页：HomePage.vue:124-126 的「开始探索」→ /courses，而 :95 的「评课中心」卡片 → /courses/reviews。路由层面 router/index.ts:331 的 /courses/:id 和 :339 的 /courses/:id/reviews 挂的是同一个 CourseDetailPage.vue，只是 titleKey 不同；站内链接也分裂：CourseListPage.vue:336 用 /courses/:id/reviews，CourseCard.vue:3、CourseListItem.vue:3、TeachingHubPage.vue:184 和 :351 用 /courses/:id。

**问题**

① 用户不知道「找课」该去哪：三个页面都能找课，但能力互不相同（有的能搜不能按院系筛，有的能筛不能搜），且从首页的两个并排 CTA 会掉进两个不同的页面；② 搜索行为不一致——在 /courses 搜完刷新页面搜索词就没了、也没法分享，在 /courses/list 却可以，同一个产品里同一个动作有两种记忆行为；③ 同一个课程详情页有两个 URL，导致后退历史里出现重复条目、浏览器标题在两个 titleKey 之间摇摆、已访问链接颜色永远对不上。

**建议改法**

① 定 /courses 为唯一发现页：把 CourseListPage 的院系折叠列表合并进 TeachingHubPage（搜索 + 院系折叠 + 热门三段），/courses/list 改成 redirect: '/courses'；② TeachingHubPage 的搜索词按 CourseListPage.vue:106-158 的写法同步到 ?q=，三个页面统一「搜索词进 URL」；③ 保留 /courses/:id 为规范 URL，把 router/index.ts:339 改成 redirect: to => `/courses/${to.params.id}`，并把 CourseListPage.vue:336 的链接改成 /courses/:id；④ 首页「开始探索」与「评课中心」统一指向 /courses。

<details><summary>核验记录</summary>

所有事实成立：TeachingHubPage 的 query 来自 usePinyinSearch（:269），全文无 router.replace，刷新即丢；CourseListPage.vue:150 写 ?q=；TeacherHubPage.vue:121 也写 ?q=；ReviewPage 无搜索框；/search 独立存在（router:360-367）。HomePage.vue:124-126 「开始探索」→/courses，:95 评课中心卡片→/courses/reviews，确实并排两个 CTA 去两个页面。router/index.ts:331 与 :339 确实都挂 CourseDetailPage.vue，只有 titleKey 不同（routes.courseDetail / routes.courseReviews）。站内链接分裂也属实：CourseListPage.vue:336 用 /courses/:id/reviews，而 CourseCard.vue:3、CourseListItem.vue:3、TeachingHubPage.vue:184 和 :351 用 /courses/:id。方案③（:339 改 redirect）可行，但须保留 name: 'course-reviews'（PostReviewPage.vue:1060 按名跳转），且 tests/e2e/review-actions.spec.ts、journey-review.spec.ts 多处 goto('/courses/42/reviews') 会变成重定向，断言 URL 的用例需同步。P1 合理。

</details>

---

#### `P1` 首页统计区「全有或全无」降级，且与写死的「10000+ 条评价」自相矛盾

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/home/views/HomePage.vue`
- `clients/web/src/i18n/locales/zh-CN/home.ts`
- `clients/web/src/i18n/locales/en-US/home.ts`

**现状**

HomePage.vue:66 用 Promise.allSettled 并行请求 api.course.getStats() 和 api.review.getReviewStats()，但 :72 只要任一 rejected 就直接 throw，:81-86 把三个数字全部归零并设 statsLoadError；:156-162 用一行红字 text-danger 整块替换掉三个 CountUp。同时 :96-98 在 reviewCount === 0 时回退到 t('home.landing.features.reviewCenter.stats')，即 zh-CN/home.ts:32 写死的 '10000+ 条评价'；teacherProfile.stats（zh-CN/home.ts:38 '500+ 位教师'）则永远是写死值，从不来自数据。

**问题**

三个问题叠在一起：① 课程统计成功、评课统计失败时，本来可用的课程数也被丢弃归零；② 用户看到的降级形态是「三个数字消失 + 一行『加载失败』」，既没有重试入口，也不知道是整站挂了还是只是统计挂了；③ 最糟的是同屏自相矛盾——上方功能卡片赫然写着「10000+ 条评价」「500+ 位教师」，下方却是红字「加载失败」。写死的数字在真实数据缺失时反而更显眼，这是在给用户编造平台规模，属于可信度问题而非样式问题。

**建议改法**

① 把两个接口的结果分别落到各自的 ref，courseCount 只依赖 courseStats、reviewCount/userCount 只依赖 reviewStats，各自独立 try/catch；② 单个指标失败时该格子显示 '—' 加一个小号「重试」文字按钮（复用 CourseListPage.vue:282-288 的重试按钮样式），而不是整块换成红字；③ 删掉 reviewCenter.stats / teacherProfile.stats 这类捏造数字，改成不依赖数据的定性文案（如「真实同学评价」「按教师查看评价」）；④ statsCount 的 '{count}+ 条评价' 在精确值下会渲染成「47+ 条评价」，改成不带 + 的 '{count} 条评价'。

<details><summary>核验记录</summary>

事实全部核实通过：HomePage.vue:66 Promise.allSettled；:72 只要任一 rejected 就 throw；:81-86 三个数归零并设 statsLoadError；:157-162 用一行 text-danger 整块替换掉 v-else 里的三个 CountUp（确无重试入口）；:96-98 reviewCount===0 时回退到 zh-CN/home.ts:32 '10000+ 条评价'；:106 teacherProfile.stats 永远是 zh-CN/home.ts:38 的 '500+ 位教师'，从不来自数据；statsCount 确为 '{count}+ 条评价'（en-US/home.ts:33 同样）。方案里引用的 CourseListPage.vue:282-288 重试按钮样式确实存在，可复用。唯一问题是严重度：这是纯错误路径的降级 + 落地页营销文案，happy path 下页面完全可用、无功能阻断、无数据丢失、无 a11y 阻塞（红字块还带 role="alert"）。与同批其他 P0（认证表单输入框全站不可见）不在一个量级。

</details>

---

#### `P2` /courses/list 一次拉全量课程、默认全部院系展开、用 v-show 全量挂载 DOM

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/course/views/CourseListPage.vue`

**现状**

CourseListPage.vue:164 调 api.course.getCoursesGrouped() 一次性取回全部院系的全部课程，:122 readDepartmentGroups 把每个分组的 expanded 都设成 true，:332 展开容器用的是 v-show 而不是 v-if，:51 的拼音索引 maxResults 设为 10000。页面既没有 Pagination.vue 也没有 InfiniteScroll.vue——这两个组件只在 ReviewFeed.vue、SearchPage.vue、NotificationsPage.vue 里用。

**问题**

① 信息层级失效：进页面就是所有院系全部摊开，几十个分组、上千门课一次性铺在屏幕上，「按院系找课」这个分组结构等于白做，用户只能靠 Ctrl+F；顶部还专门放了「展开全部」按钮（:237-251），而它的初始状态就是全展开，所以这个按钮进页面就是 disabled 的死按钮；② v-show 意味着即便用户点了「收起全部」，所有课程节点仍然留在 DOM 里，收起只省了绘制没省内存和首屏挂载成本；③ 骨架屏（:276 九个 SkeletonCard）只覆盖首次加载，但真正慢的是这个不分页的全量请求本身。

**建议改法**

① readDepartmentGroups（:118-124）把 expanded 默认改成 false，只展开第一个分组或 URL query 里指定的分组，页面首屏变成一份可扫读的院系目录；② :332 的 v-show 改成 v-if（展开状态存在 departmentGroups 里，切换不会丢）；③ 搜索命中时仍按 :72-73 强制 expanded: true，保持现有搜索体验；④ 如果全量接口的响应体积确实大，改成按院系懒加载（DepartmentSidebar.vue:227-259 已有现成的按需加载 + 分类版本号防竞态实现可直接复用）。

<details><summary>核验记录</summary>

核实无误：CourseListPage.vue:164 调 api.course.getCoursesGrouped()，该接口在 clients/shared/src/api/courses.ts:24 是 `client.GET('/api/v1/course/courses/grouped', {})`，确实不带任何分页参数；:122 readDepartmentGroups 硬编码 expanded: true；:332 用 v-show；:52 maxResults 10000。展开全部按钮（:237-251）的 disabled 绑定 allExpanded（:85-88），数据回来后必然为 true，即「进页面就是死按钮」。全仓库 Pagination 只在 SearchPage.vue、InfiniteScroll 只在 ReviewFeed.vue / NotificationsPage.vue，本页两者都没用。方案③引用的 :72-73 搜索强制 expanded:true 存在；方案④引用的 DepartmentSidebar.vue:227-259 按需加载 + categoryVersion 防竞态实现也确实存在可复用。P2 合理（注意：把默认改成 false 后「收起全部」会变成新的死按钮，建议默认展开第一组）。

</details>

---

### 用户中心与通知

共 7 条：P0 0 / P1 3 / P2 4

#### `P1` 账号/设置类页面被拆成 3 个互相重叠的「首页」+ 8 个独立整页，没有任何统一导航

> 核验确认　|　工作量：L

**位置**

- `clients/web/src/router/index.ts`
- `clients/web/src/modules/user/views/UserCenterPage.vue`
- `clients/web/src/modules/user/views/ProfileSection.vue`
- `clients/web/src/modules/user/views/IdentityHomePage.vue`
- `clients/web/src/modules/user/views/AccountProfilePage.vue`
- `clients/web/src/components/layout/AppUserMenu.vue`

**现状**

同一批「账号设置」功能现在有三个入口页，内容互相重叠：(1) UserCenterPage.vue:3 顶部的 ProfileSection.vue:45-230 用 4 张状态卡展示 身份认证/学生认证/QQ/手机 并链到 /user/identity-verification、/user/student-verification、/user/qq-binding、/user/phone-binding、/user/academic-info；(2) IdentityHomePage.vue:91-152 用 10 张纯链接卡（无任何状态）再列一遍 /account/profile、/account/security、/user/authorized-apps、身份认证、学生认证、手机、QQ、学籍、/connect、/developers/apps；(3) AccountProfilePage.vue:131-162 + :397-422 第三次渲染 身份/学生/手机 的状态徽章卡。路由层面 router/index.ts:136-165 与 :388-486 把它们注册成 8 个彼此独立的整页（/identity、/account/profile、/account/security、/user/authorized-apps、/user/identity-verification、/user/student-verification、/user/phone-binding、/user/qq-binding、/user/academic-info），页与页之间只有零散的返回链接（AuthorizedAppsPage.vue:70-76 和 AccountProfilePage.vue:16-22 是 "返回 /identity"，AcademicInfoPage.vue:4-11 是 router.back()，NotificationsPage/UserCenterPage 干脆没有）。头部菜单 AppUserMenu.vue:50-127 只暴露 7 项，缺 /identity、/notifications、/user/academic-info、/user/authorized-apps、/user/phone-binding。

**问题**

用户想改一件事要先猜入口：手机绑定在 /user 的卡片里有、在 /identity 里有、在 /account/profile 里也有，但头部菜单里没有；已授权应用和学籍信息只能靠记地址或从 /identity 进。每次进二级页都是整页替换、无侧栏，改完只能靠不一致的返回控件回上一层，多改两项就要来回 4-6 次跳转。三处状态卡还各自维护一份 status→label→badge 派生逻辑（ProfileSection.vue:264-300 与 AccountProfilePage.vue:384-394 几乎逐行重复），状态口径容易漂移。

**建议改法**

收敛成一个 /settings 区：父路由 SettingsLayout（左侧固定导航 + 右侧 <router-view>，≤768px 折叠为顶部横向 tab 或二级列表页），子路由 profile / security / verification（身份+学生+学籍合并为一页分区）/ bindings（手机+QQ）/ authorized-apps / notifications；/user/reviews|votes|favorites 保留为「我的内容」独立区，不塞进设置。把 /identity、/account/* 及 /user/* 旧地址全部 redirect 到新子路由。ProfileSection 只保留头像+昵称+邮箱+一个「账号设置」入口，删掉 4 张状态卡；状态卡与 badge 派生逻辑抽成单一 useVerificationStatusCards() 供 settings/verification 一处使用。头部菜单收敛为「我的主页 / 账号设置 / 通知 / 管理后台 / 退出」5 项，其余下沉到侧栏。

<details><summary>核验记录</summary>

三处重叠逐一核实属实：UserCenterPage.vue:3 → ProfileSection.vue:45-230 的 4 张状态卡（链到 identity-verification/student-verification/academic-info/qq-binding/phone-binding）；IdentityHomePage.vue:91-152 的 10 张纯链接卡；AccountProfilePage.vue:131-162 + :397-421 第三次渲染身份/学生/手机状态徽章。路由 router/index.ts:136-167 与 :388-483 确实是彼此独立的整页（实为 9 条，非 8 条）。派生逻辑重复也属实且行号精确：ProfileSection.vue:264-300 与 AccountProfilePage.vue:383-395 近乎逐行重复。AppUserMenu.vue:46-128 确实只有 7 项，缺 /identity、/user/academic-info、/user/authorized-apps、/user/phone-binding（/notifications 另有铃铛入口，这条算不上缺口）。需更正一处细节：AcademicInfoPage 的 goBack 是 router.push('/identity')（:142-144），不是 router.back()——它与 AccountProfilePage/AuthorizedAppsPage 的去向其实一致，不一致的是控件样式（图标裸按钮 vs 带文案链接）。结论与 /settings 收敛方案仍然成立。

</details>

---

#### `P1` 通知页没给 InfiniteScroll 传 error，加载更多失败会无限重试并刷屏 toast

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/user/views/NotificationsPage.vue`
- `clients/web/src/modules/user/useNotificationsPageController.ts`
- `clients/web/src/components/common/InfiniteScroll.vue`

**现状**

NotificationsPage.vue:143-147 只传了 :loading 和 :has-more，没传 InfiniteScroll 支持的 :error / @retry（InfiniteScroll.vue:36、:51-54、:10-19 有完整的错误态 + 重试按钮）。loadMore 失败时 useNotificationsPageController.ts:320-329 只回滚 page 并 toast.error；store 的 pageHasMore 仅在成功分支赋值（stores/notification.ts:483-485），失败时保持 true。页面上的错误 EmptyState 是 v-else-if（NotificationsPage.vue:148、:158-159），列表非空时根本不会渲染。

**问题**

第 2 页起任何一次加载失败：pageHasMore 仍为 true、InfiniteScroll 的 error 仍为 false、sentinel 仍在视口内，于是 InfiniteScroll.vue:85-92 在 loading 落回 false 的瞬间又发 loadMore → 再失败 → 再 toast，形成不可退出的请求+toast 死循环，用户既看不到「加载失败，点击重试」也无法阻止。另外错误 EmptyState 的 title 和 description 都填了 t('common.loadFailed')（NotificationsPage.vue:160-161），同一句话重复两遍，真实错误信息只在 toast 里一闪而过。

**建议改法**

controller 暴露 loadMoreError ref（失败置 true，重试时清空），NotificationsPage 传 :error="loadMoreError" @retry="loadMore"，让 InfiniteScroll 停止 observer 并渲染自带的重试按钮；loadMore 失败时同时把 pageHasMore 视为暂停（或在 hasMore 计算里 && !loadMoreError）。错误 EmptyState 的 description 改为 getErrorMessage(pageFetchError) 的真实文案。

<details><summary>核验记录</summary>

死循环可复现推理成立：NotificationsPage.vue:38-42 只传 :loading/:has-more，未传 InfiniteScroll.vue 已支持的 :error(:36,:44) 与 @retry(:53)，其错误态 UI(:10-19) 因此永不渲染；controller loadMore 失败只回滚 page 并 toast(:90-93)；store 的 pageHasMore 仅在成功分支赋值(:483-485)，失败时保持 true。于是 loading 由 true→false 时 InfiniteScroll.vue:85-92 判定 hasMore && !error && sentinel 在视口内 → 再次 emit loadMore → controller 的 `if (loading || !hasMore) return` 已不拦截 → 重复同一页失败请求；useToast.ts 无去重、每次 show 都新增一条 3s toast，确实会刷屏。错误 EmptyState 是 v-else-if(:53-54)，列表非空时不渲染；其 title 与 description 也确实都填 t('common.loadFailed')(:55-56)。方案（暴露 loadMoreError、传 :error/@retry、hasMore && !loadMoreError）可直接落地。P1 合适。

</details>

---

#### `P1` 通知页筛选是客户端过滤分页数据：空态骗人，并会静默连环拉取全部历史

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/user/useNotificationsPageController.ts`
- `clients/web/src/modules/user/views/NotificationsPage.vue`
- `clients/web/src/stores/notification.ts`

**现状**

NotificationsPage.vue:117-130 提供「全部/未读/已读」三个筛选 chip，但 useNotificationsPageController.ts:287-296 的 visibleNotifications 只是对已加载的 pageNotifications 做本地 filter，切换筛选既不重置 page（useNotificationsPageController.ts:276 的 page ref 保持不变）也不重新请求；store 的 fetchPageNotifications(stores/notification.ts:461-464) 只接受 (page, pageSize)，根本没有 filter/status 参数。hasMore 在 'read' 分支直接返回 pageHasMore（useNotificationsPageController.ts:299-309）。

**问题**

用户点「已读」时，若第一页 20 条全是未读，列表立刻变空并弹出 EmptyState「你还没有通知」（NotificationsPage.vue:173-177 用的是 user.notification.empty/emptyDesc）——文案断言用户没有通知，实际只是当前已加载页里没有已读项，用户会误以为历史通知丢了。同时列表为空使 InfiniteScroll 的 sentinel 停在视口内，InfiniteScroll.vue:64-92 的 observer + loading 结束后的补触发会一页接一页自动拉取，直到把整个通知历史拉完，用户看不到任何进度语义，只有一个转圈。

**建议改法**

把筛选下推到服务端：fetchPageNotifications(page, pageSize, filter) 透传 status 参数；activeFilter 变化时 watch 触发 page=1 重新拉取并清空列表。若后端暂不支持 filter，则至少：切换筛选时重置分页并给出 filter 专属空态文案（user.notification.emptyUnread / emptyRead，附「查看全部」按钮切回 all），并在客户端过滤模式下把 hasMore 收敛为 pageHasMore && visibleNotifications.length > 0，避免空列表触发无限自动加载。

<details><summary>核验记录</summary>

逻辑链完全成立（原文行号偏移约 +105/+235，但符号定位准确）：NotificationsPage.vue:12-25 三个筛选 chip 只改 activeFilter；useNotificationsPageController.ts:52-61 visibleNotifications 仅对已加载 pageNotifications 本地 filter，全文无 watch(activeFilter)，page(:42) 不重置也不重新请求；store fetchPageNotifications(:461-464) 签名只有 (page, pageSize)，服务端 server/internal/modules/notification/handler.go:56-63 的 List 也只解析 page/pageSize，确认后端暂不支持 status 过滤（故方案的降级分支是必需的）。空态确实说谎：列表为空时走 NotificationsPage.vue:68-72 的 EmptyState（user.notification.empty = 「暂无通知」/emptyDesc），而非「本页没有已读项」。连环拉取也成立：hasMore 在 'read' 分支恒为 pageHasMore(:69-70)，列表为空使 InfiniteScroll 的 sentinel 停在视口内，InfiniteScroll.vue:64-77 的 observer 与 :85-92 的 loading 落回补触发会一页页拉到 pageHasMore 为 false 为止。P1 合适。

</details>

---

#### `P2` MyReviewsTab / MyVotesTab / MyFavoritesTab 是三份逐行重复的分页列表，数据来源还各不相同

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/user/views/MyReviewsTab.vue`
- `clients/web/src/modules/user/views/MyVotesTab.vue`
- `clients/web/src/modules/user/views/MyFavoritesTab.vue`

**现状**

三个组件（135/118/117 行）结构完全同构：loading/loadingMore/total/page/errorMessage/loadMoreError 六个 ref + loadInitial + onMounted + loadMore（MyReviewsTab.vue:81-123、MyVotesTab.vue:204-252、MyFavoritesTab.vue:332-369），模板里的骨架屏块、错误 EmptyState + 重试按钮（三处按钮 class 字符串逐字符相同：MyReviewsTab.vue:13-19、MyVotesTab.vue:148-154、MyFavoritesTab.vue:267-273）、以及「加载更多」按钮块（MyReviewsTab.vue:35-48、MyVotesTab.vue:167-180、MyFavoritesTab.vue:286-299）也是三份拷贝。数据源却不统一：Reviews/Votes 直接 api.user.getMyReviews/getMyVotes，Favorites 走 useUserStore().fetchMyFavorites（MyFavoritesTab.vue:338-346）。MyVotesTab 的空态还漏了行动按钮（MyVotesTab.vue:183-187），另外两个都有。

**问题**

任何一次列表体验调整（改成无限滚动、加 pull-to-refresh、统一重试样式、改分页大小）都要改三处且极易漏改；现在已经漏出不一致（votes 空态没有出口按钮，用户到了死胡同）。数据源混用也让「收藏取消后其它 tab 是否同步」这类行为无法统一推理。

**建议改法**

抽 usePaginatedList<T>({ fetchPage, pageSize }) composable 返回 { items, total, loading, loadingMore, error, loadMoreError, loadInitial, loadMore, remove, patch }，再抽一个 <PaginatedList> 展示组件（props: loading/error/items/total/skeletonVariant，slots: item / empty / empty-action），三个 tab 各自只剩 10-20 行的 fetch 配置与 item 插槽。顺手统一到 store 或统一到 api（建议都走 api + 由 UserCenter 侧提供 store 缓存），并给 votes 空态补上「去看看课程」出口。

<details><summary>核验记录</summary>

重复事实完全属实（行号偏移，实际为 MyReviewsTab.vue:89-123、MyVotesTab.vue:77-117、MyFavoritesTab.vue:88-116）：三者同构六个 ref + loadInitial + onMounted + loadMore；错误 EmptyState 的重试按钮 class 字符串逐字符相同（MyReviewsTab:13-19 / MyVotesTab:13-19 / MyFavoritesTab:14-20）；「加载更多」块同样三份拷贝（:35-48 / :32-45 / :33-46）；数据源确实不统一（Reviews/Votes 直连 api.user.getMyReviews/getMyVotes，Favorites 走 useUserStore().fetchMyFavorites :93/:109）；MyVotesTab.vue:48-52 空态确实缺 #action，另两个有(:56-63 / :55-62)。仅严重度需下调：这是纯可维护性重复，唯一的用户可见症状是 votes 空态缺一个出口按钮，抽 usePaginatedList + <PaginatedList> 属于重构收益而非缺陷修复，P2 更合适（补 votes 空态按钮可先单独落地）。

</details>

---

#### `P2` SSE 断线用户完全无感知：streamError 无人消费，铃铛列表只拉一次，角标与列表会长期不一致

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/stores/notification.ts`
- `clients/web/src/components/common/useNotificationBellController.ts`
- `clients/web/src/components/common/NotificationBell.vue`

**现状**

store 在 stores/notification.ts:771-781 的 source.onerror 里 setStreamError('notification SSE connection lost') 并降级为轮询，轮询连续失败 MAX_POLL_FAILURES 次后 stores/notification.ts:527-535 直接 stopPollingFallback()，实时更新彻底停止。streamError 在 stores/notification.ts:252 定义、:830 导出，但全仓没有任何 .vue / .ts 消费它（grep 结果只剩 store 自身）。铃铛侧 useNotificationBellController.ts:231、:240-247 用 hasLoadedHistory 保证 fetchBellNotifications 一个组件生命周期内只拉一次；降级轮询只更新 unreadCount（stores/notification.ts:519-524），不更新 bellNotifications。

**问题**

SSE 挂掉后用户看到的是「一切正常」：铃铛角标可能还在涨（轮询），但点开面板永远是首次加载的那几条旧数据，且没有任何刷新入口、没有「连接已断开」提示；等轮询也失败 3 次后角标直接冻结在旧值，用户以为没有新通知。整个降级链路对用户不可见，出问题时无法自救（只能刷新整页）。

**建议改法**

store 增加 connectionState 计算（'live' | 'degraded' | 'offline'，由 streamError + pollingFallbackActive 派生）并导出；NotificationBell 面板头部在非 live 时显示一行 text-xs 提示 + 「刷新」按钮（调用 fetchBellNotifications(1,5) 和 connectSSE()）；把 hasLoadedHistory 改为「每次打开面板都刷新，或 unreadCount 变化后下次打开必刷新」，保证角标与列表口径一致；轮询彻底失败时 NotificationsPage 顶部也展示同一条降级 banner。

<details><summary>核验记录</summary>

两条核心事实属实：(1) streamError 定义于 stores/notification.ts:252、导出于 :830，全仓 grep 确认除 store 自身与 stores/__tests__/notification.test.ts 外无任何 .vue/.ts 消费者；(2) 降级轮询只走 fetchUnreadCount(:519-537) 更新 unreadCount，不更新 bellNotifications——bell 列表只在 SSE 存活时由 :361-459 的 watcher 实时更新——而 useNotificationBellController.ts:35/:44-47 的 hasLoadedHistory 保证每个组件生命周期只 fetch 一次，所以降级期间角标与面板确实会长期不一致，且面板无刷新入口、无断连提示。但「实时更新彻底停止」不成立：:771-781 的 onerror 除 startPollingFallback 外还调用 scheduleReconnect(:599-614)，SSE 会以 1s→30s 指数退避持续重连，每次重连失败又重启轮询并把 consecutiveFailures 归零，网络恢复后能自愈；且 MAX_POLL_FAILURES 是 5(:18) 不是 3。剔除「彻底停止」后，这是「降级状态对用户不可见 + 铃铛列表陈旧」的体验缺口，降 P2；方案（connectionState 派生 + 面板刷新按钮 + 打开即刷新）仍适用。

</details>

---

#### `P2` loading / empty / error 在用户中心存在三套并行实现，还混用了裸色值

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/user/views/AcademicInfoPage.vue`
- `clients/web/src/modules/user/views/AccountProfilePage.vue`
- `clients/web/src/modules/user/views/AuthorizedAppsTab.vue`
- `clients/web/src/modules/user/views/StudentVerificationPanel.vue`
- `clients/web/src/modules/user/views/IdentityVerificationPage.vue`

**现状**

同一个模块里三种写法并存：(1) 共享组件派 —— AuthorizedAppsTab.vue:3-31 与三个 MyXxxTab、NotificationsPage 用 SkeletonCard + EmptyState；(2) 手搓卡片派 —— AcademicInfoPage.vue:18-50 用 bg-bg-card rounded-xl p-8 + 裸 spinner div，把 forbidden / not-found / unknown 三种态各写成一个居中文本块，StudentVerificationPanel.vue:23-27、IdentityVerificationPage.vue:23 同款自制 spinner；(3) 脉冲骨架派 —— AccountProfilePage.vue:169-174 用 h-36 animate-pulse 占位块，:27-36 又用一个 animate-pulse 小圆点 + 「加载中」文本条。颜色上 AcademicInfoPage.vue:42 用裸 text-red-500，而 MyReviewsTab.vue:36 用 token text-danger；ProfileSection.vue:290-295 的状态色也是裸 green-500/yellow-500/red-500。

**问题**

同一次会话里用户会看到三种不同形状的「正在加载」和三种不同版式的「失败/为空」，其中手搓派没有重试按钮的统一位置、没有 role="status" 之外的语义一致性（AcademicInfoPage 有 aria-busy，AuthorizedAppsTab 的骨架没有），错误态还有的能重试、有的只能返回。裸色值绕过 @theme token，暗色主题下对比度不受控。

**建议改法**

统一到共享组件：spinner 抽成 <Spinner size>（内含 role="status" + sr-only 文案），所有整页/整块加载改用 SkeletonCard 或 <Spinner>，所有失败/为空改用 EmptyState + #action 插槽（AcademicInfoPage 的 forbidden/not-found/unknown 分别映射为带不同 action 的 EmptyState）。再包一层 <AsyncState :loading :error :empty> 收敛这三态的分支顺序。裸 red-500/green-500/yellow-500 全部换成 tailwind.css @theme 里已有的 danger/success/warning token。

<details><summary>核验记录</summary>

三套写法与裸色值全部核实属实，行号也精确：AuthorizedAppsTab.vue:3-31 用 SkeletonCard+EmptyState；AcademicInfoPage.vue:18-50 是 bg-bg-card rounded-xl p-8 + 手搓 spinner，且 forbidden/not-found/unknown 各写成一个居中块；StudentVerificationPanel.vue:20-28、IdentityVerificationPage.vue:21-29 同款手搓 spinner；AccountProfilePage.vue:169-174 是 h-36 animate-pulse、:25-36 是 animate-pulse 圆点+加载文案。裸色值也属实：AcademicInfoPage.vue:42 text-red-500，ProfileSection.vue:290-295 green/yellow/red-500，而 tailwind.css @theme 里 --color-success/warning/danger 都已定义（:31-36），MyReviewsTab.vue:36 已用 text-danger。需微调一处描述：手搓 spinner 三处都带 role="status" + aria-busy + sr-only 文案，无障碍语义并不缺失，真正不一致的是视觉形态与错误态是否可重试。严重度上这是设计系统一致性问题，无功能缺陷，P2 更合适。

</details>

---

#### `P2` 三个纯透传 wrapper 页面 + 名不副实的 AuthorizedAppsTab，可直接删

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/user/views/QQBindingPage.vue`
- `clients/web/src/modules/user/views/StudentVerificationPage.vue`
- `clients/web/src/modules/user/views/AuthorizedAppsPage.vue`
- `clients/web/src/modules/user/views/AuthorizedAppsTab.vue`
- `clients/web/src/router/index.ts`

**现状**

QQBindingPage.vue 全文 7 行，只 import 并渲染 QQBindingPanel；StudentVerificationPage.vue 全文 7 行，只渲染 StudentVerificationPanel（且缩进是 4 空格 + 双引号，和相邻文件不一致）。AuthorizedAppsTab.vue 有 399 行、命名为 Tab，但全仓唯一引用者是 AuthorizedAppsPage.vue:25、:32——它从来不是 tab，UserCenterPage.vue:10-12 的三个 tab 里也没有它。

**问题**

多了两层无信息量的间接：读代码时要多跳一次才能找到真正实现，router/index.ts:463-471、:441-450 指向的「页面」其实什么都不做；AuthorizedAppsTab 的命名会让人以为它能挂进用户中心 tab 栏，实际耦合了 AuthorizedAppsPage 的整页布局假设，评审/改动时都要先证伪一遍。

**建议改法**

删除 QQBindingPage.vue / StudentVerificationPage.vue 两个 wrapper，router/index.ts:441-450、:463-471 直接 lazyLoad QQBindingPanel.vue / StudentVerificationPanel.vue（两者 standalone 默认 true，行为等价）。保留 Panel 命名不要改成 Page —— JoinStartPage.vue:105-124 以 :standalone="false" 复用它们，joinStartPage.test.ts 也按组件名 mock。AuthorizedAppsTab.vue 重命名为 AuthorizedAppsPanel.vue；其页头在 AuthorizedAppsPage.vue:3-23（非 :57-77），收敛到统一布局后再由 layout 提供。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

router 直接 lazyLoad QQBindingPanel.vue / StudentVerificationPanel.vue（同时把两个 Panel 重命名为 QQBindingPage / StudentVerificationPage），删除两个 wrapper 文件；把 AuthorizedAppsTab.vue 重命名为 AuthorizedAppsPanel.vue，或在收敛到 /settings 布局后直接把它的内容并入 AuthorizedAppsPage.vue 并删掉 AuthorizedAppsPage 里那段重复的页头（AuthorizedAppsPage.vue:57-77 的 eyebrow/title/subtitle 在统一侧栏布局下由 layout 提供）。

</details>

<details><summary>核验记录</summary>

事实属实：QQBindingPage.vue 全文 7 行、StudentVerificationPage.vue 全文 7 行（且后者 4 空格缩进 + 双引号，与相邻文件风格不一致），两个 Panel 的 standalone prop 默认为 true（QQBindingPanel.vue:180-186、StudentVerificationPanel.vue:485-490），所以 wrapper 确实零信息量，router 直接 lazyLoad Panel 行为完全等价；AuthorizedAppsTab.vue 399 行，全仓唯一引用者确为 AuthorizedAppsPage.vue:25/:32，UserCenterPage.vue:10-12 的三个 tab 中没有它。必须修正方案中的一条：不能把两个 Panel 改名为 QQBindingPage / StudentVerificationPage——modules/admission/views/JoinStartPage.vue:105-124 以 :standalone="false" :load-on-mount="false" 复用了这两个 Panel（并有 __tests__/joinStartPage.test.ts 依赖其组件名），它们是刻意设计的可复用面板，叫 Page 反而错误。另外「AuthorizedAppsPage.vue:57-77 的 eyebrow/title/subtitle」行号不存在，该页头实际在 :3-23。

</details>

---

### 认证与入群长流程

共 10 条：P0 2 / P1 5 / P2 3

#### `P0` 入群认证的按钮样式根本没生效：AdmissionPage.css 是 scoped，跨不到子组件和手机拍照页

> **已逐行复验**（报告人）　|　工作量：M

**位置**

- `clients/web/src/modules/admission/views/AdmissionPage.vue:1084`
- `clients/web/src/modules/admission/views/FreshmanMobileCameraPage.vue:43`
- `clients/web/src/modules/admission/views/FreshmanCameraFlow.vue:114`
- `clients/web/src/modules/admission/views/OldStudentVerificationFlow.vue:120`
- `clients/web/src/modules/admission/views/AdmissionPage.css:1`

**现状**

`.primary-button` / `.secondary-button` 只定义在 AdmissionPage.css，而 AdmissionPage.vue:1084 用 `<style scoped src="./AdmissionPage.css"></style>` 引入。scoped 会把选择器编译成 `.primary-button[data-v-xxx]`，只有 AdmissionPage.vue 自己模板里的节点（以及子组件根节点）带这个属性。但真正的表单动作按钮都在子组件里：FreshmanCameraFlow.vue:88/97/106/114/122（打开摄像头/拍摄/重拍/提交材料/手机扫码拍照）、OldStudentVerificationFlow.vue:28/89/120（学校 SSO / 发送验证码 / 验证邮箱）。更彻底的是 FreshmanMobileCameraPage.vue —— 它是 router/index.ts:211 注册的独立路由，整个文件没有 `<style>` 块、也从不 import AdmissionPage.css，却在 43/57/66/74/96/105 六处用了这两个 class。

**问题**

这些按钮渲染成浏览器原生 button：无背景色、无 8px 圆角、无 44px 最小点击高度、无 disabled 的 opacity 0.6、无 focus-visible 轮廓。也就是说「提交材料」「验证邮箱」「上传材料」「回到电脑端继续」这些整条入群认证链路上最关键的 CTA，在真实浏览器里是灰色小方块，和上方 AdmissionStatePanel 里样式正常的按钮（AdmissionPage.vue:35/70/99/119，那些在父模板里所以生效）视觉完全不一致。手机拍照页尤其严重：小屏上原生 button 只有 ~20px 高，主次动作（回到电脑端 / 在手机端继续）长得一模一样，用户无法判断该点哪个，且触控目标不达标。AdmissionReissueHint.vue:52 自己重新定义了一份 .secondary-button —— 说明这个坑已经被踩过一次，只是没往上游修。

**建议改法**

把 AdmissionPage.css 从组件 scoped 样式提升为模块级全局样式：删掉 AdmissionPage.vue:1084 的 `scoped`（或改成在各消费组件里 `import '../views/AdmissionPage.css'`）。更好的做法是彻底废掉这套 class，把这些按钮换成 components/ui 的 `<Button variant="default" | "outline">`（AdmissionPage.vue:346/354 的确认弹窗已经在用），一次性拿到统一的尺寸、disabled、focus-visible 和暗色支持。顺手删掉 AdmissionReissueHint.vue:52-75 和 OldStudentVerificationFlow.vue:529-533 的重复定义。改完后必须在真实浏览器（不是 jsdom 测试）跑一遍 /join 的手机拍照页确认。

<details><summary>核验记录</summary>

逐项核实全部属实。(1) clients/web/src/modules/admission/views/AdmissionPage.vue:1084 确为 `<style scoped src="./AdmissionPage.css"></style>`，全仓 grep `AdmissionPage.css` 只有这一处引用（另一处命中是 docs/internal/frontend-ui-review-2026-08-04.md 评审文档本身），没有任何全局 import。(2) `.primary-button`/`.secondary-button` 的完整定义只存在于 AdmissionPage.css:1-50（min-height:44px 在 :12，disabled opacity .6 在 :39-43，focus-visible 在 :45-50，border-radius:8px 在 :5）。(3) 我确认 src/styles/main.css 只 @import tailwind.css，src/styles/tailwind.css、index.html、public/、design-system/ 中都没有这两个 class 的定义，所以没有任何全局兜底。(4) 子组件用了却拿不到样式：FreshmanCameraFlow.vue:88/97/106/114/122 完全对得上（打开摄像头/拍摄/重拍/提交材料/手机扫码拍照），而该文件 :577 的 `<style scoped>` 只定义了 .field-label/.field-control；OldStudentVerificationFlow.vue:28/89/120 对得上，其 :507 的 scoped 块在 :529-533 只补了 `.primary-button:disabled,.secondary-button:disabled`（印证了「已被踩过一次但没修全」）。(5) FreshmanMobileCameraPage.vue 全文 423 行，`grep -n '<style'` 无任何命中，确实零 style 块、零 CSS import，却在 43/57/66/74/96/105 六处（行号逐个精确）用了这两个 class；它由 router/index.ts:205-214（import 在 :211）注册为独立路由 `/admission/freshman/camera/:token`。(6) 对照组也成立：AdmissionPage.vue:35/44/70/99/119/147 的按钮写在父模板的 `#actions` slot 里，slot 内容在父作用域编译，会带上父的 data-v 属性，所以确实只有这些生效。(7) AdmissionReissueHint.vue:52-75 确实完整重抄了一份 .secondary-button。(8) 现有测试只用 `wrapper.get('button.primary-button')` 之类的选择器在 jsdom 里跑（admissionPageStates.test.ts:646/895、cameraCapture.test.ts:619/801），不校验样式，所以无覆盖。方案可行：AdmissionPage.vue:294-365 已在用 components/ui 的 Dialog/Input/Button（Button 在 :344 variant="outline"、:352 默认），迁移路径现成。P0 合理——认证转化主链路上的全部 CTA 渲染为无样式原生 button，手机页触控目标不达 44px。

</details>

---

#### `P0` 学生认证表单的输入框没有边框、也没有可见的键盘焦点

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/user/views/StudentVerificationPanel.vue:182`
- `clients/web/src/modules/user/views/StudentVerificationPanel.vue:138`
- `clients/web/src/modules/user/views/StudentVerificationPanel.vue:267`
- `clients/web/src/modules/user/views/PhoneBindingPage.vue:90`

**现状**

StudentVerificationPanel.vue 的全部输入控件（138 学校下拉、182 学号、216 姓名、242 学校邮箱、267 邮箱验证码、317/335 其它字段）class 都是 `w-full px-3 py-2.5 bg-transparent rounded-lg ... outline-none transition-all duration-fast focus:border-primary`。Tailwind v4 的 preflight 把所有元素设成 `border-width: 0`，styles/tailwind.css 和 styles/main.css 里也没有任何全局 input 边框规则（main.css 一共只有 2 行）。所以：没有 `border` 类 → 边框宽度是 0；`focus:border-primary` 只改 border-color，宽度还是 0；同时 `outline-none` 又把浏览器默认焦点环干掉了。

**问题**

两个后果叠加：(1) 输入框在 bg-bg-card 上是完全透明的，只有一个 placeholder 悬空，用户看不出哪里能输入、字段边界在哪；(2) 键盘 Tab 过去没有任何视觉反馈，直接违反 WCAG 2.4.7 Focus Visible。这是账号级学生认证的主表单，也是 JoinStartPage.vue:105 内嵌复用的同一份表单，属于长流程里必填的一步。同一个仓库里 PhoneBindingPage.vue:90/110 用的是正确写法 `rounded-lg border border-border bg-bg-base/60 ... focus:border-primary focus:ring-2 focus:ring-primary/15`，证明这是遗漏不是设计意图。全站 `focus:border-primary` 但缺 border 宽度的地方共 24 处。

**建议改法**

把 StudentVerificationPanel.vue 中 **10 处**（138 学校下拉、182 学号、216 姓名、242 学校邮箱、267 邮箱验证码、317 学号(LDAP)、335 密码、以及手动认证分支的 365/374/394 三个字段）统一替换成 PhoneBindingPage.vue:90 的 `rounded-lg border border-border bg-bg-base/60 ... focus:border-primary focus:ring-2 focus:ring-primary/15`；顺带 IdentityVerificationPage.vue:198/217 是同一份坏 class，建议同批修。后续沉淀 @utility field-input 或改用 components/ui/input/Input.vue 作为唯一来源（全站 focus:border-primary 共 41 处，其中约 22 处缺 border 宽度；SearchPage/PostReviewPage 那批因为有 bg-bg-elevated + focus:ring-2，可见性问题弱于本条，可排后）。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 StudentVerificationPanel 里这 7 处 class 统一替换成 PhoneBindingPage.vue:90 的那一串（`border border-border bg-bg-base/60 ... focus:border-primary focus:ring-2 focus:ring-primary/15`）。更根本的是在 styles/tailwind.css 的 @theme 之外加一个 `@utility field-input`（或直接用 components/ui 的 Input 组件）作为唯一输入框样式来源，然后把剩下 17 处 `focus:border-primary` + `outline-none` 的调用点也迁过去，避免下一个人再复制出一份隐形输入框。

</details>

<details><summary>核验记录</summary>

问题真实。StudentVerificationPanel.vue:138/182/216/242/267/317/335 的 class 确为 `w-full px-3 py-2.5 bg-transparent rounded-lg ... outline-none transition-all duration-fast focus:border-primary`，无任何 border 宽度类；Tailwind v4 preflight 的 `*{border:0 solid}` 使 focus:border-primary（只改 border-color）完全无效；styles/main.css 确实只有 2 行、tailwind.css 里也没有全局 input 规则。焦点环也确实被吃掉：tailwind.css @layer base 有 `:focus-visible{outline:2px solid var(--color-primary)}`，但 utilities 层的 outline-none 层级更高，直接覆盖成 outline-style:none。表单容器是 :117 的 bg-bg-card，输入框 bg-transparent，所以在白卡上确实只剩一个悬空 placeholder。对照 PhoneBindingPage.vue:90/110 的正确写法属实。唯一需修正的是方案的覆盖面：文件里不是 7 处而是 10 处。

</details>

---

#### `P1` admission 自成一套写死浅色的视觉系统，同一个页面里混了两套按钮，暗色模式下 JoinStartPage 直接撕裂

> 核验确认　|　工作量：L

**位置**

- `clients/web/src/modules/admission/views/AdmissionShell.vue:44`
- `clients/web/src/modules/admission/views/AdmissionPage.css:22`
- `clients/web/src/modules/admission/views/AdmissionPage.vue:294`
- `clients/web/src/modules/admission/views/JoinStartPage.vue:2`
- `clients/web/src/modules/admission/views/JoinStartPage.vue:261`

**现状**

admission 的四个骨架组件全部写死十六进制浅色：AdmissionShell.vue:44-49（`linear-gradient(#f8fafc → #eef2f7)` + `color: #0f172a`）、AdmissionStatePanel.vue:69-72（`#ffffff` / `#dbe3ee`）、AdmissionProgress.vue:200-311、AdmissionPage.css 全文。主按钮是 `background:#0f172a` + `border-radius:8px`（AdmissionPage.css:21-24），而主站主按钮是 `rounded-full bg-gradient-to-br from-primary to-accent`（LoginPage.vue:24、NotFoundPage.vue:19）。JoinStartPage 又自己造了第三套 `.join-start-primary-button`（JoinStartPage.vue:261-295）。而 AdmissionPage.vue:294-365 的绑定确认弹窗用的是 components/ui 的 Dialog/Button/Input —— 同一个页面上，「开始认证」是深蓝方角原生按钮，点开后弹窗里的「确认并开始认证」是设计系统按钮，两者宽高圆角配色全不一样。

**问题**

styles/tailwind.css:5 定义了 `@custom-variant dark (&:where([data-theme="dark"], ...))`，stores/theme.ts 也在切换主题，但 admission 全模块没有一个 `dark:` 变体和一个 token 引用。最刺眼的是 JoinStartPage.vue:2 —— 外壳是 `bg-slate-50 text-slate-950` 写死浅色，里面 105 行和 120 行内嵌的是 StudentVerificationPanel / QQBindingPanel，那两个组件用的是 `bg-bg-card` / `text-text-primary` 这类会跟主题走的 token。暗色模式下就是：白底浅灰页面中间嵌了两张深色卡片，标题黑字、卡片里白字，像两个网站拼在一起。用户从主站带着暗色偏好点进认证链接，看到的是一个突然变亮的陌生页面，会怀疑是不是钓鱼站——而这恰恰是最需要建立信任的一步。

**建议改法**

分两步：(1) 立刻止血 —— 把 AdmissionShell / AdmissionStatePanel / AdmissionProgress / AdmissionPage.css / JoinStartPage 里的 `#f8fafc #ffffff #0f172a #475569 #cbd5e1 #64748b #dbe3ee` 换成对应的 `var(--color-bg-base) / --color-bg-card / --color-text-primary / --color-text-secondary / --color-border / --color-text-muted`，这些 token 在 tailwind.css:230 的 `[data-theme="dark"]` 块里已经有暗色值，换完自动支持暗色。(2) 删掉 `.primary-button` / `.secondary-button` / `.join-start-primary-button` / `.join-start-secondary-button` 四套按钮，统一换成 components/ui 的 `<Button>`（这跟第 1 条的修法是同一件事，一起做）。语义色 `#ecfdf5/#047857`（success）、`#fffbeb/#92400e`（warning）、`#fef2f2/#b91c1c`（danger）建议提到 @theme 里补成 token，AdmissionStatePanel.vue:136-154 的四个 tone 直接消费。

<details><summary>核验记录</summary>

事实全部核实无误。AdmissionShell.vue:44-49 确为 `linear-gradient(180deg,#f8fafc,#eef2f7)` + `color:#0f172a`，:61-62 是 `#ffffff`/`#dbe3ee`；AdmissionStatePanel.vue:69-72 同为 `#ffffff`/`#dbe3ee`，:136-154 正是 info/success/warning/danger 四个 tone 的写死语义色；AdmissionProgress.vue:200-311 全文写死（:271 dot `#cbd5e1`、:296-299 `#0f766e`）；AdmissionPage.css 全文写死，主按钮 :21-24 `background:#0f172a` + :5 `border-radius:8px`。对照主站：LoginPage.vue:24 确为 `rounded-full border-0 bg-gradient-to-br from-primary to-accent`，NotFoundPage.vue:19 同款渐变胶囊。第三套按钮 JoinStartPage.vue:261-296 属实。我按核验纪律去 styles/tailwind.css 的 @theme 确认了 token 真实存在：--color-bg-base(:39)/--color-bg-card(:40)/--color-text-primary(:50)/--color-text-secondary(:51)/--color-text-muted(:52)/--color-border(:57)，且 [data-theme="dark"] 块（:230）在 :248-265 有对应暗色值，@custom-variant dark 在 :5 —— 所以方案的 token 替换可落地（@theme 变量挂在 :root，scoped CSS 里 var() 正常生效）。`grep -rn 'dark:' modules/admission/views/*.vue *.css` 返回 0，`--color-`/`text-text-`/`bg-bg-` 在整个 admission 模块也 0 命中，「全模块零 token 零 dark 变体」属实。撕裂机制成立：JoinStartPage.vue:2 是 `bg-slate-50 ... text-slate-950` 写死浅色，:105 StudentVerificationPanel 与 :120 QQBindingPanel 大量使用 bg-bg-card/text-text-primary（StudentVerificationPanel.vue:34/70/90/117 等）；stores/theme.ts:42-48 把 data-theme 写到 documentElement，App.vue:28 引入 theme store 且 :40 的 layout==='none' 只跳过 AppShell 不跳过主题。补充一点（不影响结论）：join.stuhelper.com 是独立 origin，localStorage 里的显式 dark 偏好不会跨域带过去，但 stores/theme.ts:21 默认 mode='system'，配合 tailwind.css:295-297 的 prefers-color-scheme 块，系统暗色下同样触发撕裂，机制成立。P1 合理。

</details>

---

#### `P1` 写好的 OtpCodeInput 组件从来没被用过，两处验证码输入各自手搓了一个更差的

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/components/common/OtpCodeInput.vue:1`
- `clients/web/src/modules/user/views/PhoneBindingPage.vue:102`
- `clients/web/src/modules/user/views/StudentVerificationPanel.vue:261`
- `clients/web/src/modules/user/views/StudentVerificationPage.vue:1`
- `clients/web/src/modules/user/views/QQBindingPage.vue:1`

**现状**

OtpCodeInput.vue 是一个 177 行的完整六格验证码组件：粘贴自动分格（handlePaste:157）、退格先清当前格再退到前一格（handleKeydown:124-141）、左右方向键移动（143-154）、填满自动 emit complete（emitValue:84-86）、`autocomplete="one-time-code"` 触发 iOS/Android 短信自动填充（:11）、每格独立 aria-label（cellAriaLabel:76）。grep 全仓库，除了自动生成的 components.d.ts:56 之外没有任何引用点 —— 它是死代码。而实际的两个验证码输入是手搓的单个 `<input maxlength="6">`：PhoneBindingPage.vue:102-111 和 StudentVerificationPanel.vue:261-273。

**问题**

手搓那两个既没有 `autocomplete="one-time-code"`（手机收到短信/邮件验证码不会弹自动填充，用户得切出去抄），填满六位也不会自动提交（还要再点一次按钮），StudentVerificationPanel.vue:267 那个更是连边框都没有（见第 2 条）。用户在长流程里最烦躁的一步被拖长了两次交互。同时仓库里躺着一个更好的实现无人问津。另外 StudentVerificationPage.vue 和 QQBindingPage.vue 各 7 行，模板里只有一个 `<Panel />`、不传任何 props、不设 meta，是纯转发壳。

**建议改法**

照原方案把两处换成 <OtpCodeInput v-model="..." :disabled="loading" @complete="submit" />，删掉 StudentVerificationPage.vue / QQBindingPage.vue 并让 router 直指 Panel。但**去掉「在 handleInput 里显式 target.value = ''」这一步**：该 bug 不成立。Vue 3.5（本仓库 vue ^3.5.40，实际 runtime 3.5.29）对 value 是强制 patch 的——runtime-core.cjs.js:5725 `if (next !== prev || key === "value")`，且 runtime-dom.cjs.js:574 的 patchDOMProp 拿 **el.value**（DOM 实时值）与新值比较，输入字母后 el.value='a' 而新值为 ''，二者不等，Vue 会写回 el.value=''。所以非数字字符不会滞留在格子里，加这行是无效改动。真要补的是别的：OtpCodeInput 目前无 name/id、也没有 form 关联，接入时记得补 label 关联。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

PhoneBindingPage.vue:102-111 和 StudentVerificationPanel.vue:261-273 换成 `<OtpCodeInput v-model="otpCode" :disabled="loading" :aria-label="t('...')" @complete="submit" />`，顺手拿到自动填充和填满自动提交。顺带修 OtpCodeInput 一个 bug：`:value` 是单向绑定，输入非数字时 normalizeDigits 返回空数组、emitValue 后 vnode prop 前后都是 ''，Vue 会跳过 DOM 更新，那个字母会留在格子里但不进 model —— 在 handleInput 分支里显式 `target.value = ''` 兜一下。删掉 StudentVerificationPage.vue 和 QQBindingPage.vue，router 里直接指向 Panel（Panel 已有 standalone 默认 true，行为不变）。

</details>

<details><summary>核验记录</summary>

主体成立：OtpCodeInput.vue（177 行）具备粘贴分格（:157）、退格回退（:124-141）、方向键（:143-154）、填满 emit complete（:84-86）、autocomplete="one-time-code"（:11）、每格 aria-label（:76），而生产代码零引用；实际用的是 PhoneBindingPage.vue:102-111 与 StudentVerificationPanel.vue:261-273 两个手搓单 input，均无 one-time-code、无填满自动提交，后者还叠加第 2 条的无边框问题。StudentVerificationPage.vue / QQBindingPage.vue 确各 7 行纯转发，Panel 的 standalone 默认 true（:490 / QQBindingPanel.vue:186），删壳可行。但有两处需修正：(a)「除 components.d.ts:56 外零引用」不准确——components/common/__tests__/OtpCodeInput.test.ts 引用了它，准确说法是「仅生产代码未用」；(b) 方案里要求顺带修的那个 bug 不存在。

</details>

---

#### `P1` 只写 border-<色> 不写边框宽度：表单校验红框和整片输入框边框根本不渲染

> **已逐行复验**（报告人）　|　工作量：M

**位置**

- `clients/web/src/modules/review/views/SearchPage.vue`
- `clients/web/src/modules/user/views/StudentVerificationPanel.vue`
- `clients/web/src/modules/user/views/IdentityVerificationPage.vue`

**现状**

Tailwind v4 的 preflight 对所有元素设 border-width:0，border-<色> 只设 border-color。SearchPage.vue:65 的输入框 class 是 'w-full px-4 py-3 bg-bg-elevated rounded-lg ... focus:border-primary focus:ring-2'，没有 border 也没有 border-1/2；:66 的 :class 在校验失败时加 'border-danger focus:border-danger focus:ring-danger/20'。StudentVerificationPanel.vue:138/182/216/242/267/317/335/365/374/394 共 10 个 input/select/textarea 是 'bg-transparent rounded-lg ... focus:border-primary'，同样零边框宽度。IdentityVerificationPage.vue:198/217 相同写法。

**问题**

①校验错误态完全不可见：SearchPage 高级搜索填错课程名时，border-danger 只改了 border-color，边框宽度是 0，用户看不到任何红框，只能靠一行小字 helper 猜；focus:ring-danger/20 是 ring 不是 border，20% 透明度的淡红光晕在浅紫背景上几乎不可辨。②学籍认证/身份认证面板的输入框是 bg-transparent + 无边框，直接贴在卡片底色上，视觉上根本不像可输入区域——用户看到的是一排没有容器的占位文字，focus:border-primary 聚焦时也没有任何边框出现，只剩光标。这是认证这种一次性关键流程，卡住成本很高。

**建议改法**

给这些控件补上边框宽度和静息态颜色：基础 class 改成 'border border-border bg-bg-elevated ...'（StudentVerificationPanel 顺带去掉 bg-transparent，统一用 bg-bg-elevated，和 SearchPage/PostReviewPage 的输入框保持一致）；错误态改成 'border-danger'（此时宽度已由基础 class 提供）并同时把 ring 提到 ring-2 ring-danger/40 保证可见。更根本的做法是把这套输入框样式收敛成一个 .field 基础类写进 tailwind.css 的 utility 区（和已有的 .focus-ring-field 合并——它已经定义了 focus 时的 border-color + --shadow-focus-ring，但同样依赖调用方自带 border 宽度），然后 grep 全仓库把散落的 10+ 份输入框 class 串替换掉。

---

#### `P1` 摄像头流程：没开摄像头时页面上永远杵着一块黑方块，权限被拒的报错读屏不播报

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/admission/views/FreshmanCameraFlow.vue:70`
- `clients/web/src/modules/admission/views/FreshmanCameraFlow.vue:161`
- `clients/web/src/modules/admission/views/FreshmanMobileCameraPage.vue:25`
- `clients/web/src/modules/admission/views/FreshmanMobileCameraPage.vue:144`

**现状**

两个文件里的 `<video>` 都是无条件渲染，class 是 `aspect-video w-full rounded-lg bg-slate-950 object-cover`（FreshmanCameraFlow.vue:70-76、FreshmanMobileCameraPage.vue:25-31），没有任何 v-if / v-show 挂在 streamActive 上。摄像头没打开时它就是一整块纯黑矩形，没有图标、没有「点击下方按钮打开摄像头」的引导、没有取景框提示。摄像头报错走 describeCameraCaptureError 写进 errorMessage，渲染在 FreshmanCameraFlow.vue:161 `<p v-if="errorMessage" class="text-sm text-red-600">` 和 FreshmanMobileCameraPage.vue:144-149，都是裸 `<p>`，没有 role="alert" / aria-live，也没有图标或容器背景。

**问题**

(1) 用户进入新生认证 tab 第一眼看到的是一块无来由的黑屏，在手机上占掉半个首屏，不少人会以为页面加载失败或摄像头已经打开但拍不到东西。(2) 点「打开摄像头」被浏览器拒绝后，错误文字出现在表单最底部（FreshmanCameraFlow 里它在二维码区块之后），而按钮在上面 —— 移动端很可能在折叠线以下，用户点了没反应就以为按钮坏了。(3) 报错不在 live region 里，读屏用户完全不知道刚才发生了什么；对比 AdmissionStatePanel.vue:2-9 是有 role/aria-live 的，说明团队知道该怎么做，只是这里漏了。

**建议改法**

给 `<video>` 加 `v-show="streamActive"`，未开启时渲染一个同尺寸的占位块（虚线边框 + Camera 图标 + 「打开摄像头后在这里预览」）；`cameraSupported === false` 时占位块直接换成「当前浏览器不支持拍照，请用手机扫码拍照」并把二维码入口提上来。errorMessage 的 `<p>` 加 `role="alert"`，并从表单底部挪到按钮组正上方，套上 `rounded-lg border border-red-200 bg-red-50 p-3` 让它像个警示框而不是一行小红字。顺便：FreshmanCameraFlow 和 FreshmanMobileCameraPage 的 openCamera/captureMaterial/retake/previewDataURL/streamActive/cameraSupported 六段逻辑几乎逐字重复，抽成 `useCameraCapture(maxBytes)` composable，两边各省 ~60 行。

<details><summary>核验记录</summary>

核实无误。FreshmanCameraFlow.vue:70-76 的 `<video>` 无 v-if/v-show，class 逐字为 `aspect-video w-full rounded-lg bg-slate-950 object-cover`；FreshmanMobileCameraPage.vue:25-31 同样（虽包在 `pageState==='ready'` 分支里，但该分支内无条件渲染，与 streamActive 无关，与描述一致）。报错渲染：FreshmanCameraFlow.vue:161 逐字为 `<p v-if="errorMessage" class="text-sm text-red-600">`，FreshmanMobileCameraPage.vue:144-149 为 `<p v-if="errorMessage && pageState !== 'error'" class="mt-4 text-sm text-red-600">`，两处均无 role="alert"/aria-live，无容器背景。对照组属实：AdmissionStatePanel.vue:2-9 确有 :aria-live="liveMode" + :role="liveRole"（:62-63 danger→assertive/alert，其余 polite/status），说明团队有此能力只是漏了。DOM 顺序也属实：错误 `<p>` 在二维码区块（:132-158）之后，位于表单最底部。errorMessage 由 describeCameraCaptureError 写入（openCamera :287、captureMaterial :303）。重构建议成立：不存在 useCameraCapture（全仓 grep 0 命中，composables/ 下也无），FreshmanCameraFlow.vue:280-315 与 FreshmanMobileCameraPage.vue:249-305 的 captureMaterial/retake 几乎逐字相同。两点轻微保留（不改结论）：(a) `<video>` 用 v-show 而非 v-if 是必要的，因为 openCamera 依赖 videoRef 已挂载，方案正确地选了 v-show；(b) 「移动端很可能在折叠线以下」偏推测——二维码区块 v-if="handoff" 只在点过「手机扫码拍照」后才存在，常规「打开摄像头被拒」路径下报错就紧贴按钮组下方。核心的 role="alert" 缺失与黑方块问题成立，P1 合理。

</details>

---

#### `P1` 整个 admission 模块零 i18n，全部中文硬编码

> 核验确认　|　工作量：L

**位置**

- `clients/web/src/modules/admission/views/AdmissionShell.vue:7`
- `clients/web/src/modules/admission/views/AdmissionProgress.vue:91`
- `clients/web/src/modules/admission/views/AdmissionPage.vue:113`
- `clients/web/src/modules/admission/views/FreshmanCameraFlow.vue:93`
- `clients/web/src/modules/admission/views/JoinStartPage.vue:6`
- `clients/web/src/modules/admission/cameraCapture.ts:47`

**现状**

modules/admission 下 9 个 .vue 加 cameraCapture.ts，没有一个 import useI18n（grep 只匹配到 emit/$emit）。所有文案都是模板里的中文字面量：AdmissionShell.vue:7-9「入群身份认证 / 请按当前步骤完成账号绑定和学生身份认证」、AdmissionProgress.vue:91-110 四个步骤的 label 和 description、AdmissionPage.vue:113/125/136/229/244/257/272/289 的全部状态标题、AdmissionPage.vue:492-520 的 admissionStatusLabel 12 个分支、FreshmanCameraFlow.vue:6/14/32/43/93/102/119/128 的表单标签和按钮、cameraCapture.ts:47-66 的六段摄像头报错文案。同时 i18n/locales/ 下有完整的 en-US（errors.ts / user.ts 都有 admission 相关 key），stores/locale.ts 也在跑。

**问题**

语言切到 en-US 后，全站唯一一条「必须走完否则会被踢出群」的强制流程仍然 100% 中文，包括摄像头权限被拒的排障说明（cameraCapture.ts:47 那段最长、最需要看懂的文字）。留学生/交换生正好是最可能卡在这条流程上的人群。这也意味着任何文案调整都要改 9 个文件而不是一个 locale 文件。

**建议改法**

新建 i18n/locales/{zh-CN,en-US}/admission.ts，按 `admission.shell.*` / `admission.progress.steps.*` / `admission.state.<pageState>.{title,description}` / `admission.camera.*` 分组。AdmissionPage.vue:492-520 的 admissionStatusLabel 直接改成 `t('admission.status.' + pageState)`，AdmissionProgress.vue:90-111 的 steps 数组改成从 key 生成。cameraCapture.ts 是纯函数，让 describeCameraCaptureError 返回错误码枚举（'permission-denied' / 'no-device' / 'in-use' / 'overconstrained' / 'unsupported' / 'not-ready' / 'too-large'），由调用方 FreshmanCameraFlow.vue:290 和 FreshmanMobileCameraPage 去 t()。

<details><summary>核验记录</summary>

核实无误：`grep -rl "useI18n\|\$t(" modules/admission/` 零命中（10 个 .vue + cameraCapture.ts 全无）。逐点抽查：AdmissionShell.vue:7-9 标题与说明、AdmissionProgress.vue:90-111 四步的 label/description、AdmissionPage.vue:113/125/136/226/241/... 全部状态标题、AdmissionPage.vue:492-520 admissionStatusLabel 的 13 个 switch 分支、FreshmanCameraFlow.vue:6/14/32/43/93/102/119/128、cameraCapture.ts:47-66 的七段摄像头报错文案，全是中文字面量。i18n 体系确实在跑（stores/locale.ts、i18n/locales/{zh-CN,en-US} 各 11 个模块）。方案可行：cameraCapture.ts 是纯函数，改返回错误码枚举、由 FreshmanCameraFlow.vue:290/306 和 FreshmanMobileCameraPage.vue:276/295 去 t()，是干净的边界。P1 合理。

</details>

---

#### `P2` 三个错误页视觉不一致：chunk 失败页用了裸 indigo 渐变、只能刷新；404 的次要按钮 hover 边框是空操作

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/errors/views/ChunkErrorPage.vue:13`
- `clients/web/src/modules/errors/views/NotFoundPage.vue:26`
- `clients/web/src/modules/errors/views/ChunkErrorPage.vue:22`
- `clients/web/src/modules/auth/views/AuthCallbackPage.vue:16`

**现状**

三个错误出口各说各话：NotFoundPage.vue:19 主按钮是 `bg-gradient-to-br from-primary to-accent`（token），ChunkErrorPage.vue:13 是 `bg-gradient-to-br from-indigo-500 to-violet-500`（Tailwind 原始色板，绕过 @theme），AuthCallbackPage.vue:16 是 `bg-text-primary text-bg-base ... hover:bg-accent` 且圆角是 `rounded-sm` 而另外两个是 `rounded-full`。动作数量也不一致：404 给了「回首页 + 返回上一页」两条路，chunk 失败页只有一个「重新加载」——加载失败往往就是刷不出来，用户点几次刷新之后无路可走；auth 回调失败只有「返回登录」。此外 NotFoundPage.vue:26 的返回按钮 class 里有 `hover:border-text-primary` 但整串没有 `border` 类，Tailwind preflight 下 border-width 是 0，hover 什么也不会发生（同样的写法在 StudentVerificationPanel.vue:7、QQBindingPanel.vue:6 的返回按钮上又各出现一次）。ChunkErrorPage 的 script（22-30 行）只有 useI18n，没调 updatePageMeta，而 NotFoundPage.vue:62-67 调了。

**问题**

用户在同一个产品里撞到三种错误，会看到三种不同的按钮语言，判断不出哪些是「同一个系统给我的提示」。chunk 失败是最需要逃生口的场景却给的选择最少。404 的次要按钮因为没边框，看起来就是一段灰色文字而不是可点的按钮，hover 也毫无反馈，用户根本不会去点它。ChunkErrorPage 不改 document.title，浏览器标签页还停留在上一个页面的标题上，用户在多标签场景里找不到出问题的那个页。

**建议改法**

不要引入 ShadcnButton。1) ChunkErrorPage.vue:13 把 `from-indigo-500 to-violet-500` 直接换成 `from-primary to-accent`，与 NotFoundPage.vue:19 对齐；AuthCallbackPage.vue:16 的 `rounded-sm` 改 `rounded-full`、主按钮同样换成 `bg-gradient-to-br from-primary to-accent text-white`。2) 次要/幽灵按钮统一补边框宽度：NotFoundPage.vue:26、QQBindingPanel.vue:6、StudentVerificationPanel.vue:7、IdentityVerificationPage.vue:7、InfiniteScroll.vue:14、DraftIndicator.vue:26、NotificationsPage.vue:31 全部加 `border border-border`（--color-border 在 tailwind.css @theme 已定义，dark 分支也有覆盖），hover 才有反馈。3) ChunkErrorPage 补一个 `<a :href="accountCenterURL('/')">` 的次要出口，并补 updatePageMeta({ title: t('errors.loadError.title') })——理由不是「标题停在上一页」，而是标题会误显示成加载失败的那个业务页标题，用户分不清这页已经坏了。4) 若确实想收敛成组件，应扩展 components/ui/Button.vue 增加 outline 变体（用 var(--color-border)/var(--color-text-primary)），而不是复用 slate 硬编码的 ShadcnButton。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把三个页面的按钮统一成 components/ui 的 `<Button variant="default">` / `<Button variant="outline">`，ChunkErrorPage.vue:13 的 `from-indigo-500 to-violet-500` 换成 `from-primary to-accent`。ChunkErrorPage 补一个次要按钮「返回首页」（accountCenterURL('/')），并补 updatePageMeta。NotFoundPage.vue:26 加上 `border border-border`（或直接换成 outline Button），同一处修 StudentVerificationPanel.vue:7 和 QQBindingPanel.vue:6。三个页面的插图尺寸也对齐一下：404 是 120px svg，chunk 是 64px。

</details>

<details><summary>核验记录</summary>

事实基本属实：ChunkErrorPage.vue:13 确为 `bg-gradient-to-br from-indigo-500 to-violet-500`（绕过 @theme），NotFoundPage.vue:19 用 `from-primary to-accent`，AuthCallbackPage.vue:16 用 `bg-text-primary text-bg-base ... rounded-sm hover:bg-accent`，三者按钮语言确实不一致；ChunkErrorPage.vue:11-17 只有一个 reload 按钮，而 router/index.ts:83-88 显示它只在「已重载过仍失败」后才渲染，所以「唯一动作是再刷新一次」的逃生口批评成立；NotFoundPage.vue:26 class 串确无任何 border 宽度类，tailwind.css:2 的 `@import "tailwindcss"` 带 preflight（`border: 0 solid`），hover:border-text-primary 确实是空操作，同款写法我 grep 到 7 处（QQBindingPanel.vue:6、StudentVerificationPanel.vue:7、IdentityVerificationPage.vue:7、InfiniteScroll.vue:14、DraftIndicator.vue:26、NotificationsPage.vue:31、NotFoundPage.vue:26），发现只列了 2 处属于少报不是错报。但有两点需要修正：(1) 「浏览器标签页还停留在上一个页面的标题上」不成立——router/index.ts:543-553 的 beforeEach 已按目标路由 meta.titleKey 调 updatePageMeta，ChunkErrorPage 顶替的是目标路由组件，标题显示的是目标页标题而非上一页；(2) 原方案的 `<Button variant="default">/<Button variant="outline">` 不可直接落地：全局自动注册的 `Button`（components.d.ts:18）指向 components/ui/Button.vue，其 variant 只有 primary|secondary|ghost（Button.vue:2-3），传 default/outline 会生成无样式的 btn-default；带 default/outline 的是 components/ui/button/ShadcnButton.vue，而它的 variantClasses（:25-31）写死 bg-slate-950 / border-slate-200 / bg-white，既不走 @theme token 也不跟随 [data-theme=dark]，照方案改反而把 token 化倒退成另一套硬编码色。

</details>

---

#### `P2` 步骤指示器：已完成和进行中同色，没有「第几步/共几步」，且只覆盖了三条长流程里的一条

> 核验确认　|　工作量：M

**位置**

- `clients/web/src/modules/admission/views/AdmissionProgress.vue:296`
- `clients/web/src/modules/admission/views/AdmissionProgress.vue:35`
- `clients/web/src/modules/admission/views/JoinStartPage.vue:78`
- `clients/web/src/modules/admission/views/FreshmanMobileCameraPage.vue:2`

**现状**

AdmissionProgress.vue:296-299 把 `--complete` 和 `--current` 的圆点写成同一个 `background: #0f766e`，两者唯一区别是 `--current` 的 label 变成 #0f766e（301-303 行）。`<ol>`（:35）里的 `<li>` 没有 `aria-current="step"`，也没有任何「1/4」「第 2 步」的文字，step 状态纯靠颜色传达。覆盖面上：AdmissionProgress 只在 AdmissionPage 里 v-if="session" 渲染（AdmissionPage.vue:7-12）；JoinStartPage.vue:78-96 自己做了另一套三格 nav（active 用 `border-slate-900 bg-slate-50`，done 用 emerald 文字）；FreshmanMobileCameraPage 从 loading→ready→uploaded→desktop/mobile 五个状态一个进度指示都没有。

**问题**

扫一眼看不出「我走到哪了」——已完成和当前是同一个绿点，四格全绿和三格绿一格灰在余光里几乎一样。色盲用户和读屏用户拿不到任何步骤状态（dot 是 aria-hidden，label 只有纯文本）。跨页面还有第二重不一致：同一个产品里相邻的两条准备流程（/join 起步页 和 /join/:code 认证页）用了两套长得完全不同的步骤条（一个是四格圆点条，一个是三张卡片），用户在两页之间跳转时会觉得进度被重置了。手机拍照页则完全没有上下文，用户不知道传完照片之后还有几步。

**建议改法**

AdmissionProgress.vue:296-311 改成三态可区分：complete 用实心对勾图标 + `--color-text-muted` 描边，current 用 #0f766e 实心 + 外发光，pending 用 #cbd5e1 空心。summary 区（:11）在阶段名旁边补一行「第 {{ currentIndex + 1 }} 步 / 共 4 步」。`<li>` 上加 `:aria-current="step.state === 'current' ? 'step' : undefined"`，并给每个 li 补一个 sr-only 的状态文本（已完成 / 进行中 / 未开始 / 失败）。然后把 JoinStartPage.vue:78-96 那套 nav 删掉，改成复用 AdmissionProgress（传一个 steps 变体），手机拍照页顶部也挂一个精简版（拍照 → 上传 → 选择继续端）。

<details><summary>核验记录</summary>

逐条核实属实。AdmissionProgress.vue:296-299 确为 `.admission-progress__step--complete .admission-progress__dot, .admission-progress__step--current .admission-progress__dot { background: #0f766e; }` —— complete 与 current 的圆点同色；:301-303 确实只有 `--current .admission-progress__label { color: #0f766e }` 这一处区别（也是纯色差）。`<ol>` 在 :35，`<li>` 在 :36-48 无 aria-current；圆点 :43 是 `aria-hidden="true"`；summary 区（:8-33）只渲染 `currentStep.label` 与截止时间，全文无「第 N 步」「/共 4 步」文本，我确认 steps 数组（:90-111）为 4 项而模板未输出序号。「四格全绿」也成立：renderedSteps（:128-137）在 approved 时全部置 complete，而 currentIndex=3（:116 projectionPending/approved）时前三格 complete + 第四格 current，两种情况下四个点全是 #0f766e。覆盖面属实：AdmissionProgress.vue:3 自带 `v-if="session"`，仅由 AdmissionPage.vue:7-12 的 `#progress` slot 渲染；JoinStartPage.vue:78-96 确为自造三格 nav，:86 active 用 `border-slate-900 bg-slate-50 text-slate-950`、:92 done 用 `text-emerald-700`；FreshmanMobileCameraPage.vue 的 pageState（:200-213）有 loading/ready/uploaded/desktop/mobile/expired/error，模板 :14-142 无任何进度指示。方案可落地（currentIndex 在 :113 已存在，step.state 在 :135 已有 complete/current/pending/failed 四态）。P2 合理。

</details>

---

#### `P2` 认证走完之后是死路：approved / pendingReview / JoinStart ready 三个终态都没有下一步动作

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/admission/views/AdmissionPage.vue:240`
- `clients/web/src/modules/admission/views/AdmissionPage.vue:225`
- `clients/web/src/modules/admission/views/JoinStartPage.vue:127`

**现状**

AdmissionPage.vue:240-247 的 approved 态是一个没有 `#actions` slot 的 AdmissionStatePanel，只有一句「群内禁言会由机器人自动解除。」。225-232 的 pendingReview 态同样没有任何 slot，文案写「页面会在可见时自动检查最新状态」但页面上看不到任何轮询迹象（轮询逻辑在 779-822 行，带退避，但完全静默，没有「上次检查 xx:xx」也没有手动刷新按钮）。JoinStartPage.vue:127-139 的 ready 态也只有一段说明文字，没有任何链接。对比 ProjectionPendingNotice.vue:31 是有 retry 按钮的，AdmissionPage.vue:249-282 的 invalid/expired 态也有 AdmissionReissueHint 给下一步。

**问题**

认证页跑在独立的 join 域名上（router/join-domain.ts、NotFoundPage.vue:49 的 isJoinAdmissionHost 说明这是个独立入口），意味着用户走完全流程之后停在一个没有导航栏、没有任何链接的白页上，唯一出路是自己关标签页。pendingReview 更难受——「等待审核」没有预计时长、没有可见的刷新动作、没有超时兜底提示，用户只能反复手动刷新页面，或者干脆离开而错过状态变化。

**建议改法**

保留三处终态出口，去掉重复的 deadline 展示，并修正刷新按钮的接线：
1) approved 态（AdmissionPage.vue:240-247）补 `#actions`：一个指向 `accountCenterURL('/')`（utils/redirect.ts:148，同 NotFoundPage.vue:46 用法）的主按钮 +「可以回到 QQ 群了」提示。JoinStartPage.vue:127-139 的 ready 态同样补一个回账号中心的链接。
2) pendingReview 态（AdmissionPage.vue:225-232）补 `#actions` 放「立即刷新」按钮，但**必须调 refreshPendingReviewAfterBrowserReturn()（AdmissionPage.vue:804）而不是 refreshPendingReviewState()（:809）**——前者会先 clearPendingReviewRefresh() 再发起请求，直接调后者会与已挂起的退避 timer 并发。同时 `pendingReviewRefreshInFlight`（:469）目前是普通 `let`、非响应式，按钮要显示 disabled/loading 必须先改成 ref，否则模板绑定不会更新；顺带在成功分支记录一个 `lastCheckedAt` ref 用于「上次检查 HH:mm」。
3) **删掉**原方案里「在 description 补 manualReviewDeadlineAt 预计时长」这一项——AdmissionProgress.vue:15-25 + :148-150 已经在面板正上方渲染「审核处理截止：<time>（剩余 X 小时）」并每 60 秒刷新，再加一份是同屏重复。真正的缺口是 `manualReviewDeadlineAt` 为 null 时（api.ts:261 按 nullable 解析）整条截止信息都不显示，此时才需要在面板里给一句兜底文案（如「通常在 24 小时内处理完成」）。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

approved 态加 `#actions`：一个指向账号中心的主按钮（用 utils/redirect.ts 里已有的 accountCenterURL，NotFoundPage.vue:48 就是这么用的）+ 一句「可以回到 QQ 群了」。pendingReview 态加 `#actions` 放一个「立即刷新」按钮（直接调用已有的 pendingReview 刷新函数，AdmissionPage.vue:810），并在 description 里补上 session.manualReviewDeadlineAt 推出的预计时长（AdmissionProgress.vue:148 已经算过这个 deadline，把它显示到面板里）；轮询时在面板角落显示「上次检查 HH:mm」。JoinStartPage ready 态加一个回主站的按钮。

</details>

<details><summary>核验记录</summary>

核心问题成立但「现状」有一处事实错误，且方案里有一项是重复劳动。成立的部分：AdmissionPage.vue:240-247 的 approved 态 AdmissionStatePanel 确实是自闭合、无 #actions/#help slot，只有 description「群内禁言会由机器人自动解除。」；:225-232 的 pendingReview 同样无任何 slot；JoinStartPage.vue:127-139 的 ready 态只有文案无链接；对照组也对——ProjectionPendingNotice.vue:26-34（emit 在 :31）确有「重新检查状态」按钮，AdmissionPage.vue:249-282 的 invalid/expired 确有 AdmissionReissueHint。轮询确实静默：schedulePendingReviewRefresh(:777)/pendingReviewRefreshDelay(:790)/refreshPendingReviewState(:809)，退避表在 :433，全程无 UI 反馈、无手动刷新入口。独立域名也属实：router/join-domain.ts:1-10 定义 join.stuhelper.com + /start、/verify/:code、/admission/freshman/camera/:token，三条路由 meta 均为 layout:'none'（router/index.ts:195/203/213），AdmissionShell.vue 模板 :1-33 无任何链接。utils/redirect.ts:148 的 accountCenterURL 存在，NotFoundPage.vue:43/46 正是这么用的。**错误的部分**：「pendingReview…没有预计时长」不成立。AdmissionProgress 由 AdmissionPage.vue:7-12 渲染在 AdmissionShell.vue:26 的 #progress slot 中、位于状态面板正上方；AdmissionProgress.vue:148-150 对 pendingReview 返回 {label:'审核处理截止', value: manualReviewDeadlineAt}，模板 :15-25 已渲染格式化时间 + `（剩余 X 小时 Y 分钟）`（formatRemaining :185-196），并由 :167-172 的 60 秒定时器持续刷新。所以截止时间和倒计时今天就已经显示在同屏，方案里「把 AdmissionProgress.vue:148 的 deadline 显示到面板里」会在一屏上重复渲染同一个值。严重度 P2 维持不变。

</details>

---

### 资源共享与开放平台

共 10 条：P0 0 / P1 3 / P2 7

#### `P1` 资源上传：无进度、无 accept、10MB 限制只在提交后才校验，大文件会静默卡死

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/resource/views/ResourceEditPage.vue`
- `clients/web/src/modules/resource/resourceForm.ts`

**现状**

ResourceEditPage.vue:18 定义 `MAX_RESOURCE_UPLOAD_SIZE = 10MB`，但 validateSelectedFile（:120-131）只在 submitForm 里、且在 `saving = true`（:146）和元数据校验之后才跑（:155）。文件输入（:316-320）是 `<input type="file" class="sr-only">`，没有 `accept`、没有 `multiple` 说明，外层 `<span>`（:308-321）画成虚线拖拽框但只在 hover 变色，没有任何 dragover/drop 处理，也没有 focus-within 样式。选中后只显示文件名（:311），不显示体积。提交时 resourceForm.ts:55-65 走 `readFileAsDataURL`（:108-123）把整个文件读成 data URL 塞进 JSON body，全程只有按钮上一个转圈图标（:345）。

**问题**

四个连锁问题：① 用户选了 200MB 的视频，页面毫无反应，要把标题/描述/标签/绑定全填完点提交才被告知"文件大小必须大于 0 且不超过 10 MB"；② 没有 accept，系统文件选择器不做任何过滤，选到后端不支持的类型也要等提交后才知道（且报错还被上一条 finding 里的通用文案盖掉）；③ 10MB 文件 base64 后是 13MB 的 JSON 字符串，FileReader 读取期间主线程会明显卡顿，接着是一段没有百分比、没有取消按钮、没有"正在上传"字样的纯转圈，慢网下用户根本分不清是卡住还是在传；④ 画成拖拽区却不能拖，键盘 Tab 到 sr-only input 时外框没有任何视觉变化。

**建议改法**

拆成两步。立刻能做（纯前端、零后端依赖）：把 validateSelectedFile 的体积校验提取成纯函数，在 handleFileChange(:99-103) 里即时执行，超限即清空 selectedFile 并在虚线框内红字提示，同时在 :311 文件名旁显示 formatFileSize(file.size)（可直接复用 ResourceDetailPage.vue:64-77 已有的 formatFileSize，抽到共用模块）；给 :308 的 span 加 @dragover.prevent/@drop.prevent 共用同一个 handler；加 has-[:focus-visible]:border-primary has-[:focus-visible]:ring-2 让 sr-only input 获焦时外框可见。accept 改为只做体验提示、不做白名单——要么不加 accept，要么加宽松的提示性 accept 并明确注释后端无类型白名单（service.go:226-283），绝不要据此在前端拦截。需要后端配合、单开 issue：真正的上传进度必须先由服务端提供 multipart 端点，前端再用 XMLHttpRequest.upload.onprogress（现有 fetch 客户端做不到），在那之前退而求其次——base64 编码阶段显示'正在处理文件'、请求阶段显示不确定态进度条 + 文案，至少让用户能区分卡死与在传；失败后表单已保留（:161-166 只置 errorMessage、不 reset），补一个'重试上传'按钮复用已 read 好的 dataURL，避免重复 base64。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 validateSelectedFile 提到 handleFileChange（:99-103）里立刻执行，选中即校验体积和类型，超限时清空 selectedFile 并在框内红字提示；同时把 `resource.form.fileHint`（"单个文件最大 10 MB"）之外再显示已选文件的实际体积。给 input 加 `accept`（后端允许的 MIME 列表，抽成常量与 hint 同源）。给外层 span 加 `@dragover.prevent/@drop.prevent` 走同一个 handler，并加 `has-[:focus-visible]:border-primary has-[:focus-visible]:ring-2` 让键盘可见。上传改用 FormData + axios `onUploadProgress`，按钮旁显示进度条和百分比、失败时保留已填表单并给"重试上传"按钮（当前失败后表单还在，但没有任何重试入口，用户只会再点一次提交重新 base64 一遍）。

</details>

<details><summary>核验记录</summary>

现状描述逐行核对全中：ResourceEditPage.vue:18 MAX_RESOURCE_UPLOAD_SIZE = 10*1024*1024；validateSelectedFile 在 :120-131，只在 submitForm 里被调用，且调用点 :155 位于 validateMetadata(:137) 与 saving.value = true(:146) 之后；文件输入 :316-320 是 `<input type="file" class="sr-only" @change="handleFileChange">`，确无 accept；外层 span :308-321 画虚线框但 class 只有 hover:border-primary/50，无 dragover/drop 监听、无 focus-within/has-[:focus-visible]（全仓 grep has-[ / focus-within 在 .vue 里零命中，Tailwind 4.3 支持 has-* 变体，:308 用到的 border-border-light / border-primary 在 styles/tailwind.css @theme 里都有定义，:58 与 :18）；选中后只回显 selectedFile?.name(:311) 不显示体积；resourceForm.ts:55-65 buildCreateResourcePayload → readFileAsDataURL(:108-123) 整包读 data URL 进 JSON；提交期间只有 :345 一个 animate-spin 图标。四条连锁问题成立，尤其'填完 6 个字段才被告知文件超限'是主流程可用性缺陷，维持 P1。但方案有两处不可落地：(1) 'axios onUploadProgress'——package.json 里没有 axios，前端走的是 api/client.ts 自研 fetch 客户端（:346 fetchWithTimeout/AbortController），fetch 没有上传进度 API；且后端 handler.go:34 createResource 收的是 JSON dataBase64（model.go:41-42），改 FormData 需要服务端新增 multipart 端点，不是前端一处改动。(2) 'accept = 后端允许的 MIME 列表'——服务端没有白名单：service.go:226-262 decodePayload 用 http.DetectContentType 嗅探，只要求声明类型与嗅探类型兼容（resolveResourceContentType :258-283 还专门放行 zip 容器与 legacy Office），任意类型都可上传；手写一份 accept 白名单反而会挡掉后端本来允许的类型。

</details>

---

#### `P1` 资源新建/编辑页离开无未保存提醒，填了半小时的表单一键蒸发

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/resource/views/ResourceEditPage.vue`

**现状**

ResourceEditPage.vue 全文没有 `onBeforeRouteLeave`、没有 `beforeunload` 监听（grep 零命中），也没有任何 dirty 状态跟踪。页面顶部有"返回我的资源"RouterLink（:181-187）、底部有"取消"RouterLink（:334-339），两个都是即点即走。表单本身有 6 个字段 + 一个文件（:236-330），其中 bindingsText 是多行手写文本（:295-300）。

**问题**

这是全站字段最多的表单之一，绑定关系还要按 `type: value` 一行行手敲。用户误点顶部返回、点浏览器后退、或者刷新，所有输入连同已选文件一起没了，没有任何提示、没有草稿。评课模块有草稿机制，资源模块一点都没有，同类操作两种预期。

**建议改法**

加一个 `isDirty` computed（对比 form 快照 + selectedFile 是否为 null），配 `onBeforeRouteLeave` 弹确认（复用 ui/dialog 而不是 window.confirm）和 `beforeunload` 兜刷新/关标签页；提交成功后先置 dirty=false 再 router.push，避免跳详情页时被自己拦截。底部"取消"按钮同样走这个确认。进一步可对新建模式把 form 快照写 sessionStorage，返回时提示"恢复上次未完成的上传"。

<details><summary>核验记录</summary>

通读 ResourceEditPage.vue 全文 353 行，确实没有 onBeforeRouteLeave、没有 beforeunload、没有任何 dirty 跟踪；全仓 grep onBeforeRouteLeave/beforeunload 只命中 auto-imports.d.ts:63 和 PostReviewPage.vue:450/713/939/965，资源模块零命中。顶部'返回我的资源'RouterLink 在 :181-187，底部'取消'RouterLink 在 :334-339，两个都是无拦截直接跳走；表单字段区 :236-330 确为 title/description/category/visibility/tagsText/bindingsText 六项 + 文件输入，bindingsText 多行 textarea 在 :295-300，且 resourceForm.ts:84-106 要求逐行手敲 `type: value` 否则整体抛错，输入成本高属实。对照组也成立：PostReviewPage.vue:933-939 有 handleBeforeUnload + window.addEventListener('beforeunload')（:713 在 onBeforeUnmount 里移除），:965-967 onBeforeRouteLeave 返回 confirmLeaveWithDraft() 异步等待自定义弹窗决策，配合 draftStore 自动存草稿（reviewDraftBehavior.test.ts 有覆盖），同类操作两种预期属实。方案可行且与仓库既有实现同构：ui/dialog/ 组件齐备（Dialog/DialogContent/DialogHeader/DialogTitle/DialogFooter），'提交成功后先置 dirty=false 再 router.push'也是必须的——submitForm :151 与 :160 成功后都会 router.push，不先解除会被自己的守卫拦下。唯一需要在实施时注意的小限制：sessionStorage 只能恢复 form 六个字段，selectedFile 是 File 对象无法序列化，恢复提示的文案不能写成'恢复上次未完成的上传'，应写成'恢复上次填写的资料（需重新选择文件）'；这只影响方案里标注为'进一步可'的可选项，不影响主体结论。

</details>

---

#### `P1` 轮换出来的 client secret 没有复制按钮，也没有"关掉就再也看不到"的警告

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/open-platform/views/DeveloperAppsPage.vue`
- `clients/web/src/modules/open-platform/components/ConnectEndpointsPanel.vue`
- `clients/web/src/i18n/locales/zh-CN/developer.ts`

**现状**

DeveloperAppsPage.vue:24-48 是轮换成功后的横幅：标题 `{app} 的新 client secret`（zh-CN/developer.ts:49），下面一个 `<code>`（第45-47行）裸展示 secret，右上角一个纯文字"关闭"按钮 `@click="rotatedSecret = null"`（第37-43行）。整块没有任何复制按钮、没有 secret 只显示一次的二次提醒、clientID（第33-35行）同样不能复制。"只显示一次"这句话只出现在轮换前的确认弹窗描述里（developer.ts:45 `rotateSecretDialogDescription`），横幅本身不重复。

**问题**

这是全站唯一一次能看到明文 secret 的机会。用户在弹窗里读到的"只显示一次"和真正看到 secret 之间隔了一次网络请求和一次页面滚动，横幅上唯一显眼的按钮恰恰是"关闭"——误点即永久丢失。要保存只能手动框选 `<code>` 里的长串（`break-all` 换行后手机上几乎选不准），而同一个模块的 ConnectEndpointsPanel.vue:30-38 已经有现成的复制按钮 + Check 图标反馈 + toast。丢失后唯一补救是再轮换一次，而每次轮换会立刻作废旧 secret，等于让线上应用二次中断。

**建议改法**

保留主体方案：把 ConnectEndpointsPanel.vue:59-88 的复制逻辑（copiedKey + 1.6s Check + toast）抽成 components/common/CopyableCode.vue，横幅内 secret 与 clientID 各用一个，secret 那行做成带文字的主按钮；横幅内补一行 danger 语气提示，复用 developer.ts:45 的措辞。三处修正：(1) 删掉'二次中断'的论证，改用真实理由——approveApp 是 admin-only，轮换横幅是开发者取得明文 secret 的唯一界面，丢失后必须再轮换一次、期间线上集成持续不可用；(2) 不要把'关闭'按钮做成'未复制不许关'——ConnectEndpointsPanel.vue:71 已经证明 navigator.clipboard 可能不存在（非 HTTPS 上下文直接抛错走 copyFailed 分支），硬门槛会把用户彻底锁死；改为关闭按钮降为次级样式 + 未复制过时弹一次确认即可，或加'我已保存'勾选（勾选不依赖剪贴板）；(3) 去掉'确认注册接口是否返回首发 secret'这条 TODO，已核实 service.go:681 注册返回空 secret，无需处理。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把 ConnectEndpointsPanel.vue:59-88 的复制逻辑（copiedKey + 1.6s Check 反馈 + toast 成功/失败）抽成 `components/common/CopyableCode.vue`，横幅里 secret 和 clientID 各用一个；secret 那行的复制按钮做成主按钮（带文字"复制 secret"，不只是图标）。横幅内新增一行 danger/warning 语气的提示，复用 `rotateSecretDialogDescription` 的措辞："该 secret 只显示这一次，关闭后无法再次查看"。"关闭"按钮改为次级样式并在未复制过时加一次 confirm（或直接禁用直到点过复制/勾选"我已保存"）。顺带在 submitApp（第1298-1329行）后确认注册接口是否也返回首发 secret——如果返回，现在被整个丢弃了，同样要走这个横幅。

</details>

<details><summary>核验记录</summary>

事实全部核对无误。DeveloperAppsPage.vue:24-48 确为轮换成功横幅：标题绑定 t('developer.apps.secretRotated')（zh-CN/developer.ts:49 = '{app} 的新 client secret'），clientID 裸文本在 :33-35，secret 裸 <code> 在 :45-47，右上角唯一按钮是 t('common.actions.close')（common.ts:23 = '关闭'）@click="rotatedSecret = null"（:37-43）。grep 'clipboard|copiedKey' 在整个 DeveloperAppsPage.vue 零命中，确实没有任何复制入口；'只显示一次'只出现在 developer.ts:45 rotateSecretDialogDescription。ConnectEndpointsPanel.vue:30-38 + 59-88 的复制按钮/Check 反馈/toast 也确实存在，抽公共组件可行（components/common/ 下无 CopyableCode.vue）。全仓 grep clientSecret 只有 DeveloperAppsPage.vue:1428 一处渲染明文，且 approveApp（首发 secret 的唯一来源，handler.go:116）挂在 RegisterAdminRoutes 的 admin 组下，开发者本人拿不到——所以轮换横幅确实是开发者取得 secret 的唯一界面，问题真实。降级理由：secret 是可选中文本且横幅不自动消失（手动 Ctrl+C 可用），紧邻的确认弹窗已经提示过'只显示一次'，且丢失后重新轮换并不造成'二次中断'——旧 secret 在第一次轮换时就已作废，丢新 secret 时线上应用本来就已经断了，finding 里'二次中断'的论证不成立。属于必须尽快修的可用性缺陷，但不构成 P0 阻断。另核实：submitApp（:1298-1329）丢弃 registerApp 响应不构成额外损失——service.go:657-681 buildNewApp 返回 `return app, "", nil`，注册时 ClientSecretHash 为空、ClientSecret 为空串，registeredAppResponse 的 `omitempty` 使其根本不下发。

</details>

---

#### `P2` /connect 是个只包了一个面板的冗余路由，且两个 OAuth 页有 ~120 行复制粘贴外壳

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/open-platform/views/ConnectPage.vue`
- `clients/web/src/modules/open-platform/views/DeveloperAppsPage.vue`
- `clients/web/src/modules/user/views/IdentityHomePage.vue`
- `clients/web/src/modules/open-platform/views/ConsentPage.vue`
- `clients/web/src/modules/open-platform/views/ProfileCompletionPage.vue`

**现状**

ConnectPage.vue 共 48 行，正文只有 `<ConnectEndpointsPanel />`（:34）加一个页头和两个跳转链接（:17-31，其中一个就跳 /developers/apps）。而 DeveloperAppsPage.vue:686 已经把同一个 ConnectEndpointsPanel 渲染在右栏。IdentityHomePage.vue:141 和 :147 把 /connect 和 /developers/apps 并列成两个入口。另一处：ConsentPage.vue:6-36 的 loading/error 块、:172-200 的 token/actingUser/expiresAtText computed、:237-243 的 redirectToURL，与 ProfileCompletionPage.vue:6-37、:172-184、:220-227 几乎逐行相同。

**问题**

开发者在身份中心看到"接入端点"和"开发者应用"两个入口，点进第一个发现内容是第二个的一个子区块，多一次无效跳转和一次认知消耗；这个面板的 `identityIssuer` 还在 ConnectPage.vue:47 和面板内部 :62 各算一遍。OAuth 两屏的重复外壳则意味着改一次加载动画/错误态要记得改两个文件，实际已经开始漂移（ConsentPage 有拒绝按钮，ProfileCompletionPage 没有）。

**建议改法**

拆成两半执行。
A. 去重但保留公开入口：删掉 ConnectPage.vue:47 的重复 identityIssuer（面板 ConnectEndpointsPanel.vue:62 自己已经算了），把 :11-13 的 subtitle 并入面板内部；IdentityHomePage.vue:141 的 /connect 卡片删掉，只留 :147 的“开发者应用”。但 **不要**把 router/index.ts:169 的 /connect 直接 redirect 到 requiresAuth 的 /developers/apps——若坚持删页，需先给端点参考另找一个不带 requiresAuth 的落点（例如把 ConnectEndpointsPanel 挂到一个公开的 /developers 文档路由），否则保留 /connect 路由本身、仅把它瘦成“公开端点参考”一屏。utils/redirect.ts:164 的白名单跟随最终路由决定同步。
B. OAuth 外壳抽取按原方案执行，这部分无异议：抽 OAuthHandoffLayout.vue（顶部渐变条 + 卡片外壳 + loading 块 + error 块 + 页头 + actingUser/expiresAt）与 useOAuthHandoff(token, fetcher)（token 解析、redirectToURL 协议校验、submit 状态），ConsentPage/ProfileCompletionPage 只保留各自的权限列表与 footer 按钮；顺带把本批次第 1 条的“补全页缺出口”一并在共享 footer 里解决，避免再次漂移。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

删掉 ConnectPage.vue，router/index.ts:169 的 /connect 改成 redirect 到 `/developers/apps#connect`（或直接删路由并同步 utils/redirect.ts:164 的白名单），IdentityHomePage.vue:141 那条入口一并删掉，只保留"开发者应用"；ConnectPage 里那句 subtitle 文案挪到面板内。ConsentPage/ProfileCompletionPage 抽一个 `OAuthHandoffLayout.vue`（负责卡片外壳 + loading + error + 顶部渐变条 + 页头 + expiresAt/actingUser）和一个 `useOAuthHandoff(token, fetcher)` composable（负责 token 解析、redirectToURL、submit 状态），两个页面只留各自的权限列表与 footer 按钮。

</details>

<details><summary>核验记录</summary>

事实核对全部通过，但删除方案会让一个当前公开可访问的页面变成登录后才能看。

核实通过（行号几乎逐条精确）：ConnectPage.vue 共 48 行，正文只有 :34 的 <ConnectEndpointsPanel />，页头两个跳转链接在 :17-31（其中 :24-30 跳 /developers/apps）；DeveloperAppsPage.vue:686 已渲染同一个 ConnectEndpointsPanel（import 在 :943）；IdentityHomePage.vue:141 的 /connect 与 :147 的 /developers/apps 并列成两个入口；identityIssuer 在 ConnectPage.vue:47 和 ConnectEndpointsPanel.vue:62 各算一遍；router/index.ts:169 是 /connect；utils/redirect.ts:164 白名单里确有 '/connect'。OAuth 双屏重复也属实：ConsentPage.vue:6-36 与 ProfileCompletionPage.vue:6-37 的 loading/error 块结构逐行同构，两边的 token/actingUser/expiresAtText computed（ConsentPage.vue:183-200 vs ProfileCompletionPage.vue:172-184）和 redirectToURL（ConsentPage.vue:237-243 vs ProfileCompletionPage.vue:220-226）几乎完全一致。

方案缺陷：router/index.ts:169-176 的 /connect **没有 requiresAuth**，而 :179-188 的 /developers/apps 是 requiresAuth:true。ConnectEndpointsPanel.vue 的内容完全是由 VITE_SSO_URL 推导的静态端点（:62-63 buildConnectEndpoints），本来就是可公开的接入文档。直接把 /connect redirect 到 /developers/apps#connect，等于把匿名开发者能看的端点参考页塞到登录墙后面，是功能回退。

</details>

---

#### `P2` DeveloperAppsPage.vue 1987 行：8 个区块 + 4 套表单 + 1 个手写 dialog 全塞在一个文件

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/web/src/modules/open-platform/views/DeveloperAppsPage.vue`

**现状**

单文件 1987 行（模板 930 行 + script 1055 行）。模板里依次是：页头/刷新（1-22）、secret 横幅（24-48）、应用列表卡片（108-241，含状态徽章与 4 个操作按钮）、应用资料编辑表单（242-335）、回调地址区 + 变更表单（336-462）、scope 区 + 申请表单（463-592）、审计日志面板（593-656）、手写分页 nav（657-681）、ConnectEndpointsPanel（686）、创建应用表单（688-842）、Teleport 手写原因弹窗（845-928）。script 里有 4 个 reactive 表单（form/redirectChangeForm/profileForm + scopeChangeReasons，1069-1091）、9 个互不相干的 loading 标志位（1023-1035）、约 60 个顶层函数，其中 4 组 validateXxx（1680-1723）逻辑高度重合。

**问题**

任何一处改动都要在 2000 行里定位；4 个内联表单共享同一份 `withdrawingKey`/`submittingAppID` 命名空间，状态互相牵连（1140-1160 的 reasonDialogSubmitting 要 switch 四种 key 才能算出按钮该不该 disable）；手写的 Teleport 弹窗（845-928）与 `components/ui/dialog/` 并存，成为全站第三套弹窗写法；分页（657-681）也是手写，与 `components/ui/Pagination.vue`（review/SearchPage 在用）第三套并存。新人改一个按钮很容易改错另一个申请流。

**建议改法**

按 P2 技术债排期、分批做，不要一次性重排：第一批只抽最独立、外部依赖最少的三块——ClientSecretBanner.vue（24-48，正好和上一条 finding 的复制按钮改造合并做）、AppAuditPanel.vue（593-654，自带 auditLoading/auditError/auditTotal/分页）、CreateAppForm.vue（688-842，连 availableScopes computed 一起搬）；第二批再拆 DeveloperAppCard.vue 及其三个内联表单，并把列表分页/筛选抽 useDeveloperApps.ts。手写 nav（657-680）换 ui/Pagination.vue 可以立刻单独做，是零风险的独立小 PR。两点修正：(1) 把 Teleport 弹窗换 ui/dialog 定位为写法统一，不要写成无障碍修复——现有实现已复用 useDialogFocus/useBodyScrollLock；(2) 把'创建表单拆成 /developers/apps/new 子路由'从重构里剥出去单独评估——那是 UX 与路由/i18n 的行为变更，混进纯重构会让 review 无法判断回归来源。行数口径同时更正为：模板 930 行 + script 995 行 + scoped style 60 行。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

按"卡片内的每个可折叠区 = 一个组件"拆：`DeveloperAppCard.vue`（108-241，props: item，emit rotate/withdraw/edit）、`AppProfileForm.vue`（242-335）、`AppRedirectURIsSection.vue`（336-462）、`AppScopeRequestsSection.vue`（463-592）、`AppAuditPanel.vue`（593-656，自己管 auditLoading/auditError/分页）、`ClientSecretBanner.vue`（24-48）、`CreateAppForm.vue`（688-842，同时把 availableScopes 1163-1246 挪进去）。原因弹窗（845-928）换成 `components/ui/dialog` + 一个 `ReasonDialog.vue` 薄封装；分页换成 `ui/Pagination.vue`。列表分页/筛选状态抽 `useDeveloperApps.ts` composable。创建表单可进一步拆成子路由 `/developers/apps/new`（现在它在 lg 右栏、移动端被推到 10 张卡片和分页之后，要创建应用得先滚过整个列表）。拆完主文件应在 250 行内。

</details>

<details><summary>核验记录</summary>

结构事实基本准确：文件确为 1987 行，且是全仓最大的 .vue（次大 AdmissionPage.vue 1084 行）。模板边界逐个核对命中——profileForm 表单 242-334、回调地址 section 336-461、scope section 463-591、审计 section 593-654、手写 nav 657-680、ConnectEndpointsPanel 686、创建应用 section 688 起、Teleport 845-928，全部对得上。script 里 form/redirectChangeForm/profileForm 确实起于 1069、1077、1083（rotatedSecret 在 1043 可交叉验证行号），loading 标志位密集区 1023-1035 属实，顶层函数 62 个（≈60 属实），4 个 validateXxx 在 1680/1699/1707/1715，reasonDialogSubmitting 1140-1161 确实 switch 四种 key。ui/Pagination.vue 存在且只有 review/SearchPage.vue 在用，ui/dialog/ 存在且只有 AdmissionPage.vue 在用，两条'并存'也成立。两处小失真：script 实为 932-1926 共 995 行（多算的 60 行是 1928-1987 的 <style scoped>，finding 写 1055 行且没提 style 块）；'4 个内联表单共享同一份 withdrawingKey/submittingAppID 命名空间'不准确，实际有 profileSubmittingAppID / redirectSubmittingAppID / scopeSubmittingAppID / rotatingAppID 各自独立的 ref，只有三类撤回动作共用 withdrawingKey 字符串键。降级理由：这是纯可维护性债，无任何用户可见缺陷或正确性风险；那个手写 Teleport 弹窗在 :1060-1065 已经接了 useBodyScrollLock + useDialogFocus，不是无障碍问题；仓库 AGENTS.md 与 docs/guides/frontend-development.md 都没有单文件行数约定；而方案本身是一次性新增 7 个组件 + composable + 新子路由的大爆炸重构，回归面远大于收益。

</details>

---

#### `P2` ProfileCompletionPage 没有任何退出路径，用户在 OAuth 中途被关进死胡同

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/open-platform/views/ProfileCompletionPage.vue`
- `clients/web/src/modules/open-platform/views/ConsentPage.vue`

**现状**

ProfileCompletionPage.vue:117-138 的 footer 只有两个按钮："刷新"（重新拉取补全状态）和"继续"（继续授权）。整页是 `<main class="min-h-screen">` 独立布局，没有 AppHeader、没有返回链接。只有 error 分支（:31-37）里才有一个"打开身份主页"的 router-link。对照 ConsentPage.vue:130-139，同一套 OAuth 流程的授权页是有明确"拒绝"按钮的，点了会调 denyConsent 把用户带回申请方。

**问题**

用户被第三方应用带到这一屏，看到"必须补全手机号/实名"后如果不想给，界面上没有任何一个可点的出口：没有拒绝、没有取消、没有返回应用、没有回首页。唯一操作是关标签页——而申请方那边拿不到 deny 回调，会一直挂在等待态。同一个流程里前一屏（Consent）教会用户"可以拒绝"，后一屏突然收回这个选项，是明确的信任断层。

**建议改法**

只做可落地的一步：在 ProfileCompletionPage.vue:117-138 的 footer 最左侧补一个次级 router-link to="/identity"，复用 error 分支 :31-37 已有的 Home 图标 + 现成文案 key common.openPlatformProfileCompletion.openIdentityHome（'返回账号中心'），并在旁边加一句小字说明“取消后可稍后从应用重新发起授权”（新增一条 i18n key，zh-CN/en-US 各一条）。不要改按钮文案——common.ts:140-142 的 '我已补全，继续' / '重新检查' 已经表达清楚语义。若确实需要把 deny 回调传回申请方，先在 server/internal/modules/openplatform/handler.go:78 旁新增 POST /profile-completion/deny（复用 service_consent.go 的 deny 语义 + deleteProfileCompletionChallenge），再在前端接上；这属于后端先行项，不应作为本条的前置阻塞。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

footer 左侧补一个与 ConsentPage 同款的次级"取消授权"按钮，调对应的 deny/abort 接口并 redirect 回申请方（若后端暂无该接口，至少给一个"返回身份主页"的 router-link，并在文案上说明"取消后可稍后重新发起授权"）。同时把"刷新"按钮的语义讲清楚——现在用户在新标签页补完资料回来，不点刷新就还是旧状态，按钮却只写"刷新"，建议改成"我已补全，重新检查"。

</details>

<details><summary>核验记录</summary>

核心事实成立但叙事方向反了，且部分方案已实现。已核实：ProfileCompletionPage.vue:117-138 footer 确实只有 loadCompletion 和 continueAuthorization 两个按钮；router/index.ts:122-133 该路由 meta.layout='none'，所以确实没有 AppHeader/导航兜底；只有 error 分支 ProfileCompletionPage.vue:31-37 有指向 /identity 的 router-link；ConsentPage.vue:130-139 确实有 submitDecision(false) -> api.openPlatform.denyConsent（ConsentPage.vue:227）。

但三处需要修正：
(1) 流程顺序说反了。server/internal/modules/openplatform/service_completion.go:239-285 显示补全页是**前置**屏：ContinueProfileCompletion 校验通过后才 BuildConsentChallenge 并返回 ConsentURL（:282-285）。所以用户是先到补全页、后到 Consent 页，不存在“前一屏教会用户可以拒绝，后一屏突然收回”的信任断层，恰好相反。
(2) 按钮文案引用不准。i18n/locales/zh-CN/common.ts:140-142 实际是 continue:'我已补全，继续'、refresh:'重新检查'，不是发现里写的“继续”和“刷新”。因此原方案后半段“建议改成‘我已补全，重新检查’”已经是现状，属于无效建议。
(3) 主方案不可直接落地：server/internal/modules/openplatform/handler.go:77-78 只注册了 GET /profile-completion 与 POST /profile-completion/continue，没有 deny/abort 端点（对比 :74-76 consent 有 accept/deny），所以“调对应的 deny/abort 接口并 redirect 回申请方”需要先补后端。

综合：只是一个前置中转页缺少显式出口（关标签页与不点继续效果等同，后续 Consent 页仍可拒绝），不构成 P1。

</details>

---

#### `P2` ResourceListPage 手搓 loading/空态/分页，与全站三套写法并存；筛选把 bindingType/bindingValue 这种内部字段直接甩给用户

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/resource/views/ResourceListPage.vue`

**现状**

ResourceListPage.vue:347-356 用 6 个裸 `div.h-44.animate-pulse` 当骨架（DeveloperAppsPage.vue:77 用的是共享 `SkeletonCard`），:358-373 手写错误态、:375-386 手写空态（共享 `components/common/EmptyState.vue` 有 icon/action 插槽却没用），:429-450 是"加载更多"按钮（review/SearchPage 用 `ui/Pagination.vue`，DeveloperAppsPage.vue:657-681 又是手写 nav）。筛选表单（:262-339）是四个并排的自由文本框：搜索、标签、"绑定类型"、"绑定值"，后两个没有下拉候选、没有说明。列表按 updatedAt 展示（:424）但没有任何排序控件。

**问题**

① 空态（:375-386）只有标题和描述，没有出口：用户输了搜不到的关键词，页面告诉他"没有结果"，但"清除筛选"的 X 按钮在页面最上方筛选栏里（:328-337），移动端已经被四个 11 单位高的输入框顶出屏幕，用户只能自己往回滚或手动删字。② "绑定类型/绑定值"是数据模型概念，学生不可能知道要输 `course` 还是课程 ID，等于两个永远空着的输入框，还白占了移动端一屏。③ 同一个 App 里骨架屏、空态、分页各三套写法，视觉细节（圆角、图标尺寸、间距）各不相同。

**建议改法**

1) loading（:347-356）换 components/common/SkeletonCard.vue，与 CourseListPage.vue:276、DeveloperAppsPage.vue:77 对齐。2) 错误态（:358-373）与空态（:375-386）换 components/common/EmptyState.vue，用 #icon/#action 插槽给出口：hasFilters（:39）为真时 action 是“清空筛选”（直接调已有的 clearFilters）；为假时 action 是“发布资料”（RouterLink to name:'resource-new'）。3) 分页**不要**统一到 ui/Pagination.vue——它全仓零引用、是死代码，要么先删要么单独立项；本页保持“加载更多”按钮即可，或统一到已在用的 components/common/InfiniteScroll.vue（ReviewFeed.vue:62、NotificationsPage.vue:38 已是它的使用者），二选一。4) bindingType 目前后端无枚举（service.go:375 仅 TrimSpace），所以别硬编前端 select；先把这两个字段折进默认收起的“高级筛选”，移动端首屏只留搜索 + 标签，等后端给出可枚举的 binding 类型再改 select+联动 picker。5) 排序控件确认需要后端先给 sort 参数（api.gen.ts:7859-7870 当前无），列为后续项。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

loading 换 `SkeletonCard`，错误/空态换 `EmptyState` 并通过 `#action` 插槽给出"清除筛选"（`hasFilters` 为真时）和"去上传第一个资源"（无筛选时）两种不同的空态与出口。绑定类型改成 select（枚举后端支持的 type：课程/教师/学校），绑定值随类型联动成 picker；或整体折进一个默认收起的"高级筛选"，移动端首屏只留搜索框。分页统一到 `ui/Pagination.vue` 或统一到 `common/InfiniteScroll.vue`（已存在但没人用），三套择一。排序控件需要后端补 `sort` 参数（clients/shared/src/api/resource.ts:21 当前不支持），可作为后续项。

</details>

<details><summary>核验记录</summary>

行号和主体观察全部属实，但方案依赖的两条“现有组件使用情况”是反的，会误导落地。

核实通过：ResourceListPage.vue:347-356 确为 6 个裸 div.h-44.animate-pulse；:358-373 手写错误态；:375-386 手写空态且只有 emptyTitle/emptyDesc 没有任何出口；:429-450 是“加载更多”按钮；筛选表单 :262-339 确为四个并排自由文本框；:424 按 updatedAt 展示且无排序控件；清除筛选 X 在 :328-337。对照组也对：DeveloperAppsPage.vue:76-77 用 SkeletonCard + :80 用 EmptyState，:657-681 又是手写 nav 分页。排序需后端补参数也属实——clients/shared/src/types/api.gen.ts:7859-7870 listResources 的 query 只有 page/pageSize/query/tag/bindingType/bindingValue，没有 sort。

错误点：
(1) “review/SearchPage 用 ui/Pagination.vue”——错。grep 全 src，components/ui/Pagination.vue 零引用（SearchPage.vue:468/585/689 只是名为 resetSearchPagination 的本地函数）。它本身就是死代码，不能当作“统一目标”。
(2) “common/InfiniteScroll.vue（已存在但没人用）”——错。它在 components/business/review/ReviewFeed.vue:62(import :82) 和 modules/user/views/NotificationsPage.vue:38(import :84) 都在用。
(3) “后两个没有下拉候选、没有说明”——半错。i18n/locales/zh-CN/resource.ts:10 bindingTypePlaceholder='如：course、term、school'、:12 bindingValuePlaceholder='如：8、2025-2'，占位示例是有的，缺的只是下拉候选。
(4) 方案里“枚举后端支持的 type：课程/教师/学校”无依据——server/internal/modules/resource/service.go:375 只对 BindingType 做 TrimSpace，repository.go:97 直接作为 SQL 过滤参数，后端没有任何枚举校验；占位符暗示的是 course/term/school（课程/学期/学校），不是“教师”。

</details>

---

#### `P2` 删除资源用原生 window.confirm，全站唯一两处，且没说清后果

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/web/src/modules/resource/views/ResourceDetailPage.vue`
- `clients/web/src/modules/resource/views/ResourceMinePage.vue`

**现状**

ResourceDetailPage.vue:148 和 ResourceMinePage.vue:223 都是 `if (!window.confirm(t('resource.detail.deleteConfirm', { title })))` 直接 return。全 src 下 window.confirm 只有这两处（其余模块都走 components/ui/dialog 或 DeveloperAppsPage 的 Teleport 弹窗）。文案只带资源标题，不提版本、下载记录、已分享链接会怎样。

**问题**

原生 confirm 的按钮是浏览器语言（英文系统上显示 OK/Cancel，与 zh-CN 界面割裂）、无法套用主题和暗色、无法区分"删除"与"取消"的危险程度（两个按钮长得一样重），Safari/移动端还可能被"阻止此页面弹出对话框"直接吞掉——真被吞掉时 confirm 返回 false，用户点删除毫无反应，也不知道为什么。而且这是不可逆操作，弹窗却不说明影响范围。

**建议改法**

抽一个 ConfirmDeleteResourceDialog.vue 供 ResourceDetailPage.vue:144-163 与 ResourceMinePage.vue:221-239 共用，底座直接复用 components/ui/dialog（Dialog.vue + DialogContent/Header/Title/Description/Footer 已具备 title/description id 关联），而不是再手写一套；若不想引入 ui/dialog 作为第三套写法，也可对齐仓内已有的内联确认范式 ReviewCard.vue:212-243。文案层面只需**补充影响范围**（历史版本、已分享下载链接失效），不必重写“此操作不可恢复”——zh-CN/resource.ts:42/:57 已有。确认按钮用 danger 样式并写“删除资料”，取消按钮设为初始焦点（ui/dialog 可用 data-dialog-initial-focus 约定，参考 DeveloperAppsPage.vue:895），删除中的 loading 绑定到弹窗确认按钮（复用现有 deleteLoading / deletingIDs），弹窗在请求完成后再关闭。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

改用 `components/ui/dialog`，标题"删除《{title}》"，正文列出后果（"该资源的全部历史版本将一并删除，已分享的下载链接会立即失效，此操作不可撤销"），确认按钮用 danger 样式并写"删除资源"而非"确定"，取消为默认焦点。两处共用一个 `ConfirmDeleteResourceDialog.vue`。顺带把删除中的 loading 态放进弹窗按钮（现在 ResourceDetailPage 的 deleteLoading 只反映在页面按钮上，弹窗关掉后用户看不到进度）。

</details>

<details><summary>核验记录</summary>

主体事实成立，但两处描述夸大/失实，且严重度偏高。已核实：ResourceDetailPage.vue:148 和 ResourceMinePage.vue:223 确为 window.confirm；grep 全 clients/web/src 只有这两处 window.confirm。

失实点：
(1) “文案只带资源标题…弹窗却不说明影响范围”——i18n/locales/zh-CN/resource.ts:42 和 :57 实际是“确定删除“{title}”吗？此操作不可恢复。”，en-US/resource.ts:42/:57 是 'Delete "{title}"? This cannot be undone.'，已经明确告知不可逆。缺的只是“版本/下载链接”这一层影响范围，不是完全没有后果说明。
(2) “其余模块都走 components/ui/dialog 或 DeveloperAppsPage 的 Teleport 弹窗”——grep 显示 components/ui/dialog 全仓只有 modules/admission/views/AdmissionPage.vue 一处引用；评价/回复的删除确认走的是自己手写的内联面板（ReviewCard.vue:212-243、ReplyCard.vue:27-29），并非 ui/dialog。所以“全站统一走 ui/dialog”这个参照系不存在。
(3) “Safari/移动端可能被‘阻止此页面弹出对话框’吞掉”属推测——用户手势触发的 confirm() 正常不会被抑制，仅在反复弹出/跨源 iframe 场景才会。

剩下的真问题（原生按钮跟随浏览器语言、无法主题化/暗色、危险与取消按钮同权重、无 loading 反馈）都成立，但删除本身工作正常且已有确认+不可逆提示（ResourceDetailPage.vue:152-162 的 deleteLoading 也确实在页面按钮上有 spinner，见 :264-269），属一致性/打磨范畴。

</details>

---

#### `P2` 开发者控制台 4 处必填"原因"输入框只有 aria-label，屏幕上完全没有标签

> 核验确认　|　工作量：S

**位置**

- `clients/web/src/modules/open-platform/views/DeveloperAppsPage.vue`

**现状**

DeveloperAppsPage.vue:302-307（应用资料变更原因）、:430-435（回调地址变更原因）、:818-824（每个勾选 scope 的用途说明），都是 `<textarea :aria-label="t('...')" />`，既没有可见的 `<span>` 标签，也没有 placeholder，就是一个空白框。而同一文件里的原因弹窗（:886-898）是标准写法：可见 label + textarea。这些字段全是必填，validateForm/validateProfileForm/validateRedirectChangeForm（:1680-1723）会在提交时拦下并报"请填写…"。

**问题**

视力正常的用户看到的是一个凭空冒出的空白框——勾一个 scope 就多出一个不知道要写什么的框；只有提交被拒后才从错误条里反推它是必填的"用途说明"。而 scope 的用途说明是审核方判断是否批准的唯一依据，也是最终展示在 ConsentPage.vue:122-124 给终端用户看的"申请理由"，写不好直接导致审核被驳回。同一文件里两种表单标签写法并存。

**建议改法**

全部改成与弹窗（:886-898）一致的 `<label class="grid gap-1.5"><span class="text-sm font-medium text-text-secondary">{{ t('...') }}</span><textarea .../></label>`，去掉 aria-label（可见 label 已足够）。scope 用途说明那处再加一句 placeholder 说明它会被展示给终端用户，例如"例：用于在课表页展示你的头像和昵称——这段话会展示在用户的授权确认页"，并在下方标注剩余字数/最少字数要求。

<details><summary>核验记录</summary>

逐条核实无误，且数量“4 处”正确（发现正文只列了 3 处，漏写了第 4 处）。DeveloperAppsPage.vue:302-307（profileForm.reason，只有 :aria-label="t('developer.apps.profileReasonLabel')"）、:430-435（redirectChangeForm.reason）、:818-824（form.scopeReasons[option.scope]）确实都是裸 textarea：无外层 <label>、无可见 <span>、也无 placeholder（已逐个 grep -A6 确认属性列表）。正文遗漏的第 4 处是 :557-563 的 scopeChangeReasons（:aria-label="t('developer.apps.scopeChangeReasonLabel')"），同样问题，所以标题的“4 处”反而是准确的。

对照写法确实并存于同一文件：:886-897 的原因弹窗是 <label class="grid gap-1.5"><span class="text-sm font-medium text-text-secondary">{{ t('developer.apps.reasonLabel') }}</span><textarea/></label>；更刺眼的是同一个资料编辑表单里 :251-261（displayName）、:263-272（description）、:275-284（homepageURL）全都用可见 <span> 标签，唯独紧随其后的 :302 原因框没有。

必填性也核实：validateProfileForm（:1699-1705）要求 profileForm.reason.trim()，validateRedirectChangeForm（:1715-1723）要求 redirectChangeForm.reason.trim()，validateForm（:1680-1697）要求每个 selectedScope 的 scopeReasons 非空，validateScopeChangeForm（:1707-1713）同理——全部只在提交时报错。scope 用途说明最终展示给终端用户也属实：ConsentPage.vue:122-124 渲染 scope.reason（'用途：{reason}'）。i18n key 均已存在（zh-CN/developer.ts:62/70/87/92/99），改成可见 label 可直接复用，方案零风险且与文件内既有范式一致。

</details>

---

#### `P2` 资源模块 8 处 catch 全部吞掉服务端错误，403/404/413 都显示同一句"请稍后重试"

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/web/src/modules/resource/views/ResourceDetailPage.vue`
- `clients/web/src/modules/resource/views/ResourceListPage.vue`
- `clients/web/src/modules/resource/views/ResourceEditPage.vue`
- `clients/web/src/modules/resource/views/ResourceMinePage.vue`

**现状**

四个页面共 8 处 `catch (_error) { void _error; xxxError.value = t('...') }`：ResourceDetailPage.vue:157-159（删除失败）、:180-183（下载失败，固定 `resource.detail.downloadFailed`）、ResourceListPage.vue:117-127、ResourceEditPage.vue:87/113/161-163（`resource.form.saveFailed` = "资料保存失败，请稍后重试。"）、ResourceMinePage.vue:110/233。全模块没有一处引用 `@/api/errors` 的 `getErrorMessage`，而 open-platform 侧（DeveloperAppsPage.vue:937、ConsentPage.vue:162、ProfileCompletionPage.vue）全部用 `getErrorMessage(err, fallback)` 透出后端文案。

**问题**

下载是资源共享的核心动作，而 downloadResource（:165-189）无论后端返回 401 未登录、403 私有资源无权限、404 文件已删、还是 500，用户看到的都是同一句"下载失败"——正好是评审要点里问的"权限不足提示是否明确"，答案是完全不明确：用户不知道该去登录、该去申请、还是该重试。上传同理，10MB 之外的后端限制（文件类型黑名单、配额、重名）全被压成"资料保存失败"，用户只能盲改。同一个 App 里两个模块两种错误策略，也让后端写好的错误文案在资源模块全部作废。

**建议改法**

降级为 P2 并缩小范围：(1) 把 8 处改成 getErrorMessage(err, t('...')) 仍然值得做，但要在描述里写清它的真实语义——把后端错误码映射到前端 errors.ts 的本地文案（404→A0000404'请求的资源不存在'、503→B0000004'服务暂时不可用，请稍后重试'、429→A0000429），而不是透出后端 message；同时把 ResourceEditPage.vue:113 从清单里剔除，那是本地绑定解析错误，保留现有 invalidBindings 文案即可。(2) 下载路径删掉 401/403 分支，改为只加 404 一支：复用 ResourceDetailPage.vue:79-81 已有的 isResourceNotFoundError，命中时切到'资源不存在或你无权访问'空态而不是错误条（措辞要跟后端 404 防泄露的口径一致，不能说成'私有资源，请联系上传者'，那会把后端刻意隐藏的存在性泄露出去）；503/409 走可重试提示。(3) 上传失败要真正可诊断，光换 getErrorMessage 不够——contentType 与嗅探结果不符（service.go:246 ErrResourceContentTypeMismatch）现在统一是 response.BadRequest 默认码 A0000400'请求参数错误'，用户照样不知道该改什么；正解是服务端给 ErrResourceContentTypeMismatch / ErrResourcePayloadSizeInvalid 分配独立错误码并在 i18n/locales/*/errors.ts 补对应文案，前端再消费。(4) 给 ResourceDetailPage.vue:282-288 的 downloadError 补重试按钮这条保留，成本低。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

把这 8 处统一改成 `errorMessage.value = getErrorMessage(err, t('...'))`（与 DeveloperAppsPage.vue:1325 同款），保留现有 t() 作为兜底。下载路径再加一层分支：捕获 401 → 提示"请先登录"并给登录入口；403 → "该资源为私有/需要对应权限"并给"联系上传者"或返回列表；404 → 直接切到"资源不存在"空态而不是错误条。删除/保存失败的错误条补一个重试按钮（现在 ResourceDetailPage.vue:283 那条 downloadError 只是一段红字，没有任何后续动作）。

</details>

<details><summary>核验记录</summary>

catch 清单本身准确：ResourceDetailPage.vue:157-159、:180-183，ResourceListPage.vue:117-127，ResourceEditPage.vue:87/113/161-163，ResourceMinePage.vue:110/233，恰好 8 处 `catch (_error)`，且全模块 grep getErrorMessage 零命中。但问题描述有三处硬伤：(1) getErrorMessage 并不'透出后端文案'——api/errors.ts:113-126 只拿 error.code 去查前端本地 i18n 键 `errors.${code}`，查不到就返回调用方传的 fallback，永远不会返回后端的 message 字段；所以'后端写好的错误文案在资源模块全部作废'不成立，任何模块都没在用后端文案。(2) 下载路径的 401/403 分支不可达：handler.go:33 该路由挂的是 optionalAuthMW，匿名可访问，不产生 401；service.go:174-176 私有资源对非属主直接 `return ErrResourceNotFound`，handler.go:200-202 映射成 404，handler_integration_test.go:56-60 明确断言越权下载返回 StatusNotFound——后端是故意用 404 防存在性泄露，按方案加 403 分支属于永不触发的死代码。(3) 413 也不会出现：超限走 service.go:239 ErrResourcePayloadSizeInvalid → isResourceBadRequestError（:328-338）→ 400，标题里的 403/404/413 只有 404 对。(4) ResourceEditPage.vue:113 那处 catch 捕的是 parseResourceBindings 本地抛的解析异常（resourceForm.ts:93/98），根本不是服务端错误，混进清单不当。另外 finding 没提 ResourceDetailPage.vue:7 已 import getErrorStatus/isApiError，:79-81/:128-132 详情加载路径已经区分了 404 与通用失败，'两个模块两种错误策略'的对比被夸大了。剩下的真问题范围小得多：下载/删除/保存失败时 404（已删）、409（挂载禁用）、503（存储不可用，handler.go:216-236）全被压成同一句'请稍后重试'。

</details>

---

### 管理后台

共 9 条：P0 0 / P1 5 / P2 4

#### `P1` analytics 与 workspace 两个仪表盘是同一份代码的两份拷贝，指标数据完全相同

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/admin/apps/web-ele/src/views/dashboard/analytics/index.vue`
- `clients/admin/apps/web-ele/src/views/dashboard/workspace/index.vue`
- `clients/admin/apps/web-ele/src/router/routes/modules/dashboard.ts`

**现状**

dashboard.ts:21 和 :32 注册了「分析页」「工作台」两个侧边栏菜单。两个 .vue 都 import 同一个 `getAdminStats`，都复制了同一套 `fetchRequestSeq` 竞态防护、`canAccess`、`adminErrorMessage`、ElSkeleton 骨架、快捷入口列表（评课管理/举报管理/实名审核…）。analytics 的 4 个 KPI 卡是 totalReviews/pendingReports/hiddenReviews/weekReviews，workspace 的 overviewItems + queueItems 是 totalReviews/publishedReviews/totalReports/todayReviews + pendingReports/hiddenReviews/weekReviews——同一批字段换个排版。此外 analytics 是全站 20 个页面里唯一不用 AdminContentLayout 的（自己写 `.admin-dashboard__header` 的 h1+p+按钮），workspace 用了 AdminContentLayout 但正文全是硬编码中文（:48 '待处理举报'、:80 '评课总量'、:210 '处理队列'、:230 '常用入口'）。analytics 的 i18n key 还是 Vben 模板遗留的 `overview.users/visits/downloads/usage`，值却被改成了评课/举报文案。

**问题**

管理员侧边栏有两个入口，点进去看到的是同一个接口的同一批数字，只是卡片顺序和配色不同——没人能说清什么时候该看哪个。同时维护两份竞态逻辑和两套 CSS（analytics.css / workspace.css），改一个指标要改两处。analytics 不走 AdminContentLayout 导致它的标题字号、内边距、卡片边框和其余 19 个页面对不上，从任意业务页跳回首页会有明显的版式跳变。

**建议改法**

保留删除 analytics、以 workspace 为唯一首页的方向，但补齐 4 步：1) dashboard.ts 删除 Analytics 子路由的同时把 `affixTab: true` 移到 Workspace，并给父路由 /dashboard 加 `redirect: '/workspace'`；2) 同步改 preferences.ts:12 `defaultHomePath: '/workspace'`（不要依赖 route-resolution 的兜底）；3) 删除 views/dashboard/analytics/ 时一并删除 index.test.ts，并把它里面仍有价值的断言（不得回流 Vben 的 layout/notification demo locale）改写成针对 workspace 的版本，同时 zh-CN 与 en-US 两侧同时移除 admin.dashboard.analytics 的 8 个 key，避免 locales.test.ts 的双语一致性断言失败；4) 把 analytics 独有的 summaryItems 与 moderationLoad 负载条按需迁进 workspace（否则会丢掉「待处理举报/举报总量」的负载视图），并把 workspace :48/:80/:210/:230 等 14 处中文抽到 admin.dashboard.*，队列区置顶、pendingReports>0 加 warning 色标、总量卡下沉。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

删掉 analytics/，只保留 workspace/ 作为唯一首页（它的「处理队列 → 点击跳转对应页」信息层级更贴近运营动作），dashboard.ts 里移除 analytics 路由并把 `/dashboard` 默认重定向到 workspace，同时删掉 analytics.css 和 `admin.dashboard.analytics.*` 那 8 个遗留 key。保留下来的 workspace 补两件事：把 :48/:80/:210/:230 等 14 处中文抽到 `admin.dashboard.*`；把 overviewItems（总量类）和 queueItems（待办类）的层级拉开——待办队列置顶并给 pendingReports>0 加 warning 色标，总量卡下沉。

</details>

<details><summary>核验记录</summary>

事实核对基本属实：router/routes/modules/dashboard.ts:19-37 确实注册了 Analytics(/analytics) 与 Workspace(/workspace) 两个菜单；analytics/index.vue:12 与 workspace/index.vue:12 同 import getAdminStats，两边各自复制了 fetchRequestSeq 竞态防护（analytics:138-153 / workspace:129-145）、canAccess、adminErrorMessage、ElSkeleton 与快捷入口列表；workspace 硬编码中文的行号精确命中 :48 '待处理举报'、:80 '评课总量'、:210 '处理队列'、:230 '常用入口'；analytics i18n key 确为 Vben 遗留的 overview.users/visits/downloads/usage，zh-CN/admin.json:596-615 与 en-US/admin.json:596-615 各 8 个 key，值已改成评课/举报文案；我逐个检查 22 个 views/**/index.vue，未用 AdminContentLayout 的只有 analytics 与 _core/profile（Vben 核心页），业务页里 analytics 确是唯一一个。细节偏差两处（不影响结论）：analytics 第一张 KPI 的主数值是 todayReviews、totalReviews 是卡片脚注；analytics 还多了 summaryItems（:107-129）和 moderationLoad 负载条（:132-136），workspace 没有，所以「指标数据完全相同」应表述为「同源 AdminStats 的不同切片」。真正需要调整的是方案完整性：直接删 analytics/ 会踩三个坑——preferences.ts:12 `defaultHomePath: '/analytics'` 仍指向已删路由（虽有 router/route-resolution.ts:75-81 兜底到首个可达菜单，但登录落地页语义会漂）；`affixTab: true` 挂在 Analytics 路由（dashboard.ts:23），删掉后没有固定首页标签；views/dashboard/analytics/index.test.ts 是一份专门守护该目录的测试，它断言目录内 .vue 仅 index.vue、locale 的 dashboard.analytics 只含 overview、且 admin.dashboard 顶层 key 恰为 ['analytics','quickActions','summary']——删目录和删那 8 个 key 会直接让这份测试红，CI 挡住。

</details>

---

#### `P1` 入群认证策略页零数据死锁：新部署时管理员无法创建第一条策略

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`

**现状**

index.vue:319 「新增目标认证群」按钮写死 `:disabled="loading || policies.length === 0"`；创建对话框的来源下拉 createSourceOptions（:76-81）是 `policies.value.map(...)`，列表为空时下拉为空；submitCreatePolicies（:241-248）在 `!createPolicyForm.sourcePolicyID` 时直接抛错「请填写目标认证群号并选择要复制的策略」。页面主体是 `v-for="policy in policies"`（:349-352），policies 为空时整个 body 只剩 :600-602 那句「成员黑名单管理已迁移至独立页面」的灰字。

**问题**

新环境/后端返回空列表时，管理员看到的是一个几乎空白的页面 + 一个灰掉的唯一按钮，没有任何空态说明、没有引导、没有报错。整条入群认证策略功能从后台侧完全无法启动——必须绕过前端去数据库或 API 建第一条。这是唯一入口被自己锁死，属于功能不可用。

**建议改法**

分两层做，别只改前端 disabled：1) 后端先补 bootstrap 能力——在 admission 模块加「无来源创建」路径（service_queries.go:207 的校验放开为 sourcePolicyID 可空 + repository 用一组默认列值 INSERT，或新增 POST /admission/policies/bootstrap），否则任何前端改动都只是把错误提前。2) 前端立刻能做的是空态与解释：在 index.vue:350 的 v-for 外层加 `v-else`（!loading && !loadError && policies.length === 0）渲染 `<ElEmpty description="尚未配置任何目标认证群">`，并在描述里明确写出「首条策略需由运维按 production-go-live 落库/或等待 bootstrap 接口上线」，同时把 :319 的 disabled 保留但补 `title`/tooltip 说明灰掉原因，避免管理员对着灰按钮无从判断。3) 后端能力就绪后再解除 policies.length===0 限制，并把创建对话框的「复制策略」改为可选（有策略默认选第一条，无策略隐藏该 FormItem）。4) 独立可做的小修：submitCreatePolicies :253-259 的串行循环记录已成功的 guildID，失败时 `ElMessage.warning('已创建 X 个，Y 个失败：...')` 并保留对话框，不要整批静默回退。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

1) 去掉 `policies.length === 0` 的 disabled 条件，只保留 `loading`。2) 创建对话框把「复制来源」改成可选：有策略时默认选第一条并显示下拉，无策略时隐藏下拉、走后端默认值建库（若后端强制要求 sourcePolicyID，则新增一个「从系统默认模板创建」选项）。3) 在 `v-for` 外层补 `v-else` 空态：`<ElEmpty description="尚未配置任何目标认证群">` + 一个 primary 的「创建第一个目标认证群」按钮，和 header 里的按钮指向同一个 openCreatePolicyDialog。4) submitCreatePolicies 的批量循环（:253-259）当前是串行 await 且中途失败不报进度——失败时把已成功的 guildID 列出来，改成 `ElMessage.warning('已创建 X 个，Y 个失败：...')` 而不是吞掉。

</details>

<details><summary>核验记录</summary>

代码事实全部核对无误：admission-policy/index.vue:319 `:disabled="loading || policies.length === 0"`、:76-79 createSourceOptions 由 policies 派生、:241-247 无 sourcePolicyID 直接报「请填写目标认证群号并选择要复制的策略」、:351 v-for="policy in policies"、:601 空列表时只剩黑名单迁移灰字，整页确实没有 ElEmpty/v-else 空态。后端也印证了 bootstrap 缺口：server/internal/modules/admission/service_queries.go:207-211 强制 sourcePolicyID 非空，repository_queries.go:76-103 是 `INSERT ... SELECT ... FROM group_admission_policies WHERE id = $4` 的复制式创建，全仓没有第二条创建路径；migrations/000009_remove_default_school_seed.up.sql:26 还会删掉旧默认种子策略，scripts/seed.sql 仅开发用（docs/guides/database-migrations.md:16）。但两处需要修正：(1) 现状描述里「必须绕过前端去数据库或 API 建第一条」不准确——API 同样建不出来，只能落库或跑种子 SQL；(2) 因此原方案 1)+2) 的纯前端改法不可行：去掉 disabled 后管理员只会打开一个来源下拉恒为空、点确定必然报错的对话框，比现在灰按钮更糟；「走后端默认值建库」「从系统默认模板创建」在后端都不存在对应能力。前端的 disabled 其实是后端约束的忠实反映，纯 UI 层修不好，且 docs/guides/production-go-live.md:174 把「group_admission_policies 至少包含 platform=qq, guild_id=178037297」列为上线验收项，说明首条策略在运维流程里是落库步骤，故 P0（前端功能不可用）偏高。

</details>

---

#### `P1` 开放平台应用页：单行 10 个按钮塞进 460px 固定列，且全局 actionLoading 让整张表一起失能

> 核验确认　|　工作量：M

**位置**

- `clients/admin/apps/web-ele/src/views/open-platform/apps/index.vue`

**现状**

index.vue:775 操作列 `:default-width="460"` fixed=right，:777-869 里同时渲染 10 个 ElButton：按 pendingScopes 循环的「通过范围/驳回范围」、按 pendingRedirectURIRequests 循环的「通过回调/驳回回调」、通过应用、资源授权、轮换密钥、暂停、恢复、吊销。整个 1030 行文件只有 :53 一个 `const actionLoading = ref(false)`，6 处按钮全部 `:disabled="actionLoading"`，全文件只有 2 处 `:loading=`（都在对话框里）。

**问题**

1) 视觉：460px 固定右列在 1440 宽笔记本上吃掉 1/3 视口，剩余业务列被挤压；scope 是循环渲染的，一个应用申请 3 个 scope 就是 6 个按钮 + 4 个固定按钮，`admin-action-group` 的 flex-wrap 会折成 3 行，行高炸开。2) 状态反馈：点「吊销」第 3 行时，全表所有行的所有按钮同时变灰，没有任何 spinner 指示是哪一行在处理；管理员无法判断请求是否发出，容易重复点或误以为卡死。对比 identity-review 用 `reviewingActionsByUserId` 做的按行按动作追踪，这里是明显退化。

**建议改法**

1) 操作列降到 ~140px，只保留当前状态下的主操作（pending→「审核」，approved→「资源授权」），其余全部收进 `<ElDropdown>` 的「更多」，宽度回收给业务列。2) 待处理的 scope / redirectURI 不要在操作列平铺——把它们放进一个「审核」抽屉/对话框（复用 ResourceGrantsDialog 的形态），一次性列出所有待批项并逐条通过/驳回。3) 把 `actionLoading: ref<boolean>` 换成 `actionByAppID: reactive<Record<string, ActionName>>`，按钮改 `:loading="actionByAppID[row.id]===\'suspend\'"` / `:disabled="Boolean(actionByAppID[row.id])"`，只锁当前行。

<details><summary>核验记录</summary>

逐条核对属实：open-platform/apps/index.vue:771-775 操作列 `fixed="right"` + `:default-width="460"`，且 PersistentAdminTableColumn.vue:32 确认 defaultWidth 在无持久化宽度时直接作为列宽生效；:778-908 的 admin-action-group 内确有 10 个 ElButton（pendingScopes 循环的通过/驳回范围、pendingRedirectURIRequests 循环的通过/驳回回调、通过应用、资源授权、轮换密钥、暂停、恢复、吊销），PersistentAdminTable.vue:252-257 的 .admin-action-group 是 `display:flex; flex-wrap:wrap; gap:8px`，多 scope 时确实会折行撑高行；:53 全文件唯一 `const actionLoading = ref(false)`，8 个处理函数（:101/120/140/240/259/283/308/328）共用它，模板里 10 处 `:disabled="actionLoading"`（794/806/824/834/850/864/874/884/894/904），没有任何按行按动作的 loading，对照 users/identity-review/index.vue:58 的 `reviewingActionsByUserId` + :265/:269 的 userReviewing/userActionLoading 确实是退化。仅两处措辞偏差且都属低估而非错报：实际是 10 处 disabled 而非「6 处」；两处 `:loading=` 在 :602（错误 Alert 的重试按钮）和 :619（表格级 loading）而非「都在对话框里」——结论（无逐行动作反馈）不受影响。方案（收窄操作列 + ElDropdown 更多 + 待审项收进对话框 + actionByAppID 逐行状态）可落地，且与 identity-review 现有模式一致。

</details>

---

#### `P1` 用户系统 4 个页面完全没接 i18n，locale 里根本没有对应 key

> 核验确认　|　工作量：M

**位置**

- `clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`
- `clients/admin/apps/web-ele/src/views/users/member-blacklist/options.ts`
- `clients/admin/apps/web-ele/src/views/users/admission-sessions/options.ts`
- `clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue`
- `clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json`

**现状**

zh-CN/admin.json 的 `users` 段只有 identityReview / schoolConfig / studentVerification / systemConfig 四个子键，admissionPolicy、admissionSessions、memberBlacklist、freshmanVerification 一个 key 都没有。对应地：admission-policy/index.vue 全文 650 行只有 2 处 `$t`、72 行含中文（:51-67 的 policyFieldLabels、:70-74 的策略选项、:312-313 页面标题描述、:322 按钮文案、:586 「保存影响」）；member-blacklist 的 4 个子组件 + options.ts 全部 0 个 `$t`（options.ts:20-41 的原因/来源/动作枚举全是中文字面量）；admission-sessions 同样 0 个 `$t`。content/operation-logs 只有 3 个 `$t`，7 个列头（:94 '时间'、:103 '管理员'、:115 '动作'、:124 '资源'、:136 '变更前'、:146 '变更后'、:156 '请求'）硬编码——而 `admin.common.time` = '时间' 这个 key 本来就存在。zh/en 两侧 498 个 key 完全对齐，说明 i18n 机制本身是健康的，纯粹是新页面没接。

**问题**

切到 en-US 时，用户系统四个页面（含被当作重构范本的 member-blacklist、admission-sessions）整页仍是中文，只有外框标题变英文，出现中英混排。更麻烦的是这几个恰好是最近新写/刚拆分过的页面——说明「拆组件」这一轮把 i18n 漏掉了，后面照着抄的页面会继续漏。

**建议改法**

1) 在 zh-CN/en-US 的 admin.json `users` 段补 `admissionPolicy`、`admissionSessions`、`memberBlacklist`、`freshmanVerification` 四个子树，把上述字面量搬进去；枚举类（options.ts 的 reasonCode/source/action 映射）改成存 i18n key 再 `$t(key)`，不要存中文值。2) operation-logs 的 7 个列头直接换成已有的 `admin.common.time` / 新增 `admin.content.logs.*`。3) 加一条 lint 拦截：eslint 规则 `@intlify/vue-i18n/no-raw-text`（或简单的 CI grep：`grep -rP '(label|title|placeholder|description)="[\x{4e00}-\x{9fa5}]' src/views/`），否则下一个新页面还会漏。

<details><summary>核验记录</summary>

逐项核对全部成立。(1) locales/langs/zh-CN/admin.json 的 users 段只有 identityReview/studentVerification/schoolConfig/systemConfig 四个子键（用 python 解析确认），admissionPolicy/admissionSessions/memberBlacklist/freshmanVerification 一个都没有；zh/en 各 498 key、对称差集为 0，与描述完全一致。(2) users/admission-policy/index.vue 共 650 行，只有 :304 和 :335 两处 $t；`grep -cP '[\x{4e00}-\x{9fa5}]'` 精确返回 72 行含中文；policyFieldLabels 在 :50-68、joinHandlingStrategyOptions 在 :70-74、页面标题/描述在 :312-313、按钮「新增目标认证群」在 :322、「保存影响」在 :586 —— 全部对得上。(3) member-blacklist 的 BlacklistFilters/BlacklistTable/CreateBlacklistDialog/ReleaseBlacklistDialog + options.ts 的 $t 计数全部为 0；options.ts:19-65 的 SOURCE_LABELS/CREATED_FROM_LABELS/RELEASE_REASON_OPTIONS/STATUS_OPTIONS/SCOPE_OPTIONS 确为中文字面量。(4) content/operation-logs/index.vue 只有 3 处 $t（:65/:72/:82），7 个列头硬编码在 :94/:103/:115/:124/:136/:146/:156，与描述行号逐一吻合；admin.common.time='时间' 确实已存在。(5) 额外佐证严重度：packages/@core/preferences/src/config.ts:142 widget.languageToggle=true，语言切换器是开着的，en-US 用户能真实切到。两处小瑕疵不影响结论：admission-sessions/index.vue 其实有 2 处 $t（不是「0 个」），且 admin.content.logs 下已存在 admin/action/targetType/targetId 四个 key，方案里「新增 admin.content.logs.*」有一部分可直接复用；另可注意 member-blacklist/index.vue:193 与 admission-policy/index.vue:313 把页面标题写死，而 admin.routes.userSystem.memberBlacklist / admissionPolicy 早已存在，属于「有 key 不用」。

</details>

---

#### `P1` 用户系统页面用硬编码 Tailwind slate/bg-white 绕开 Element 变量，暗色主题下白底黑字

> 核验确认　|　工作量：S

**位置**

- `clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`
- `clients/admin/apps/web-ele/src/views/users/member-blacklist/BlacklistTable.vue`
- `clients/admin/apps/web-ele/src/views/users/admission-sessions/AdmissionSessionTable.vue`
- `clients/admin/apps/web-ele/src/views/users/member-blacklist/ReleaseBlacklistDialog.vue`

**现状**

admission-policy/index.vue 有 22 处硬编码色值类：:353 `rounded border border-slate-200 bg-white p-5 shadow-sm`（策略卡片外壳）、:357 `border-b border-slate-200`、:361 `text-slate-900`、:372/:378 `text-slate-500`/`text-slate-600`、:558 `text-slate-900`、:585 `text-slate-500`。BlacklistTable.vue:62/:86 `text-xs text-slate-500`、:149 `text-slate-400`；AdmissionSessionTable.vue:82/97/109/113/125/232 同款；ReleaseBlacklistDialog.vue:78 `text-slate-600`。全 views 目录 `dark:` 变体命中数为 0，而 Vben 5 的 preferences 默认支持 `mode: 'dark'`。其余 16 个页面（AdminContentLayout、PersistentAdminTable、content/*、open-platform/*）一律用 `var(--el-bg-color)` / `var(--el-text-color-primary)` / `var(--el-border-color)`。

**问题**

开启暗色主题后，入群认证策略页的每张策略卡片是纯白背景配 slate-900 深色文字，钉在深色 shell 里，边框也是浅灰——整页刺眼且与顶栏侧栏割裂；黑名单/会话表格里的次要文字用 slate-500 压在深色行背景上，对比度不足到基本读不出来。浅色主题下也有问题：slate-200 和 Element 的 `--el-border-color` 不是同一个灰，卡片边框与相邻表格边框深浅不一致。

**建议改法**

这 4 个文件把颜色类全换成 Element 变量：`bg-white`→`background: var(--el-bg-color)`，`border-slate-200`→`var(--el-border-color)`，`text-slate-900`→`var(--el-text-color-primary)`，`text-slate-600/500/400`→`var(--el-text-color-regular)` / `var(--el-text-color-secondary)`。布局类（grid/flex/gap/p-5/text-xs）保留 Tailwind 无妨。更省事的做法：把 admission-policy 那张卡片抽成 `shared/AdminCard.vue`（复用 AdminContentLayout 里已有的 header/body 样式约定），策略页和后续页面直接用它，杜绝再写一次 bg-white。

<details><summary>核验记录</summary>

事实与因果链都成立，我一度怀疑「换成 el 变量也修不了暗色」，实测后被推翻。(1) 类名核对：admission-policy/index.vue 的 slate- 出现次数 grep -c 精确为 22，且 :353 `rounded border border-slate-200 bg-white p-5 shadow-sm`、:357、:361、:372、:378、:558、:585 与描述逐行对上；BlacklistTable.vue:62/:86 text-slate-500、:149 text-slate-400；AdmissionSessionTable.vue:82/97/109/113/125/232；ReleaseBlacklistDialog.vue:78 text-slate-600 —— 全部命中。views 目录 `dark:` 变体计数为 0。构建产物 dist/css 里 .text-slate-500{color:var(--color-slate-500)}、.bg-white{background-color:var(--color-white)} 确实生成，不是无效类。(2) 暗色确实是默认态：packages/@core/preferences/src/config.ts:126 theme.mode='dark'，apps/web-ele/src/preferences.ts 没有覆盖 theme。(3) 关键验证：dist 静态 CSS 里 --el-bg-color 只有 #fff、没有 element-plus dark/css-vars.css，我原以为 el 变量在暗色下也不会变；但 apps/web-ele/src/app.vue 调用了 useElementPlusDesignTokens()，packages/effects/hooks/src/use-design-tokens.ts:184 把 --el-bg-color 映射到 --background、:187 --el-border-color→--border、:312 --el-text-color-primary→--foreground，而 packages/@core/base/design/src/design-tokens/dark.css 在 .dark 下把 --background 改成深灰、--foreground 改成 0 0% 95%。所以走 var(--el-*) 的页面（AdminContentLayout.vue:49/56/57、PersistentAdminTable.vue）真的会跟随暗色，而写死 bg-white/text-slate-900 的这 4 个文件不会——「白卡片钉在深色 shell 里」「slate-500 压在深色行背景上」两条症状均成立，原方案（换 Element 变量）也确实能修好。

</details>

---

#### `P2` 三个巨型页面（1030/787/743 行）仍是单文件，member-blacklist 的拆分范式没有推广

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/admin/apps/web-ele/src/views/open-platform/apps/index.vue`
- `clients/admin/apps/web-ele/src/views/authorization/grants/index.vue`
- `clients/admin/apps/web-ele/src/views/users/identity-review/index.vue`

**现状**

member-blacklist 已经拆成 index.vue(258) + BlacklistFilters.vue(104) + BlacklistTable.vue(165) + CreateBlacklistDialog.vue(154) + ReleaseBlacklistDialog.vue(121) + options.ts(115)，admission-sessions 同构。而 open-platform/apps/index.vue 仍是 1030 行（虽已外提 ImportCasdoorAppDialog / ResourceGrantsDialog，但 :97-367 的 8 个 lifecycle handler、:377-414 两个 prompt、:421-508 的 tag/label/can* 判定、:523-557 的 toolbar、:559-600 的密钥告警条、13 个表格列全在一个文件）；grants/index.vue 787 行（:45-62 的 role 常量、:103-144 的 label 映射、:339-401 的 6 个筛选控件、:617-670 的创建对话框全内联）；identity-review/index.vue 743 行（:480-633 的证据详情对话框 + :635-662 的驳回对话框全内联）。

**问题**

三个页面各自都超出了一屏能理解的范围，同一个 `adminErrorMessage` / `resetPageAndFetch` / `handleActionError` 在每个文件里被重抄一遍（全 views 目录 `adminErrorMessage` 出现在 20 个文件）。改一处 tag 配色或一个筛选行为要在四五个文件里同步。新人照着最近的页面抄，抄到的是巨型单文件而不是 member-blacklist 的范式。

**建议改法**

顺序反过来：先解契约、再拆文件，且把 adminErrorMessage 抽取从本条里摘出去单独决策。第 1 步——把这 5 个 *-load-errors.test.ts 从「按文件路径做源码字符串断言」改成按目录聚合断言（例如允许 src/views/<模块>/**/*.vue 任一文件命中，而不是钉死 index.vue），或者干脆换成挂载组件的行为测试（member-blacklist/subcomponents.test.ts、admission-sessions/subcomponents.test.ts 已经是这种写法，可直接照抄），否则任何拆分都会红。第 2 步——按原文的目录形态拆 open-platform/apps、authorization/grants、users/identity-review，其中 grants 没有被 load-errors 源码快照覆盖，风险最低，建议作为第一个样板；apps 和 identity-review 放在契约改造之后。第 3 步——adminErrorMessage 的去重是独立议题：当前 23 份复制是被 5 个测试主动要求的，要动就得先让团队明确废掉「每个视图必须内联 adminErrorMessage」这条约定，再一次性迁到 shared/errors.ts 并删掉对应断言；不要把它塞进拆分 PR 里顺手做。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

按 member-blacklist 的目录形态逐个拆，每个页面产出 5 个文件：
• open-platform/apps/ → AppFilters.vue（:523-557 toolbar）、AppTable.vue（表格 13 列）、AppRowActions.vue（:771-869 操作列 + 上一条建议的 dropdown 收敛）、IssuedSecretAlert.vue（:559-600 密钥一次性展示 + 复制）、useAppLifecycle.ts（:97-367 的 8 个 handler + 按行 loading map）、options.ts（:421-457 的 appStatusTag/scopeStatusTag/statusLabel/scopeStatusLabel 映射）。index.vue 目标 ≤250 行。
• authorization/grants/ → GrantFilters.vue（:339-401）、GrantTable.vue、CreateGrantDialog.vue（:617-670）、options.ts（:45-62 roleOptions/assignableRoleOptions/sectionRoles + :103-144 的 roleLabel/stateLabel/sourceLabel/stateTagType/scopeLabel）。
• users/identity-review/ → IdentityFilters.vue、IdentityTable.vue、EvidenceReviewDialog.vue（:480-633，顺带把驳回理由内联进来）、options.ts（docTypeLabel/verifyMethodLabel/statusTag）。
同时把 20 个文件里重复的 `adminErrorMessage` 提到 `shared/display.ts` 旁边的 `shared/errors.ts`，一次导出。

</details>

<details><summary>核验记录</summary>

事实基本精确，但方案与仓库里既有的测试契约正面冲突，照做会红 CI。事实核对：wc -l 确认 open-platform/apps/index.vue=1030、authorization/grants/index.vue=787、users/identity-review/index.vue=743；member-blacklist 的 258/104/165/154/121/115 六个文件行数完全吻合；引用的区间也对（apps 的 handleApproveScope 起于 :97、handleActionError :372、promptLifecycleReason :377、resetPageAndFetch :416、appStatusTag :421、#toolbar :523、issuedSecret ElAlert :559；grants 的 roleOptions/assignableRoleOptions/sectionRoles :45-62、roleLabel :103、#toolbar :348、创建对话框 :617；identity-review 的证据对话框 :480、驳回对话框 :635）。两处需修正：一是 'adminErrorMessage 出现在 20 个文件' 实测是 23 个 .vue 各自定义（含测试文件共 28 个文件提及）；二是更要命的——这份重复是被测试主动钉死的架构契约：content/content-management-load-errors.test.ts:38、open-platform/open-platform-load-errors.test.ts:35、users/admission-management-load-errors.test.ts:29、users/verification-load-errors.test.ts:29、_core/profile/profile-load-errors.test.ts:26 都在断言 expect(source).toContain('function adminErrorMessage(error: unknown)')，把它提到 shared/errors.ts 会直接打穿 5 个测试文件覆盖的十几个视图。同理，verification-load-errors.test.ts:69-96 对 identity-review/index.vue 逐字断言 ':disabled="detailTargetReviewing()"'、':loading="detailTargetActionLoading('approve')"' 等，把证据对话框搬进 EvidenceReviewDialog.vue 就会失败；open-platform-load-errors.test.ts:77-91 也对 apps/index.vue 断言 '@change="resetPageAndFetch"'、'@current-change="fetchData"'，外提 toolbar/分页同样会失败。

</details>

---

#### `P2` 分页控件 4 套 layout 各写各的，11 个页面根本没有每页条数选择

> 核验确认　|　工作量：S

**位置**

- `clients/admin/apps/web-ele/src/views/shared/AdminContentLayout.vue`
- `clients/admin/apps/web-ele/src/views/users/identity-review/index.vue`
- `clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue`
- `clients/admin/apps/web-ele/src/views/authorization/grants/index.vue`
- `clients/admin/apps/web-ele/src/views/users/member-blacklist/BlacklistTable.vue`

**现状**

15 个带分页的页面出现 4 种不同的 ElPagination 配置：`total, prev, pager, next`（11 个页面：content/reviews:305、reports:337、teachers:345、sensitive-words:355、open-platform/apps:919、consents:330、audit-events:329、token-probe-evidence:394、users/identity-review:475、student-verification:392、freshman-verification:377）；`prev, pager, next, sizes, total`（operation-logs:174，total 被放到最后）；`total, prev, pager, next, sizes`（member-blacklist/BlacklistTable:159、admission-sessions/AdmissionSessionTable:254）；`total, sizes, prev, pager, next, jumper` + `:page-sizes="[10,20,50,100]"`（grants:609-613，唯一带 jumper 的）。AdminContentLayout 只提供了一个 `#pagination` 插槽，不约束内容。

**问题**

同一个后台里分页条位置和能力四种形态：11 个页面的管理员被钉死在默认 pageSize，处理 200 条待审举报时只能一页页翻，无法一次看 100 条；只有 grants 能直接跳页；total 有时在最左有时在最右，眼睛每次都要重新找。这也是最容易在下一个新页面继续走样的地方。

**建议改法**

在 shared/ 下加 `AdminTablePagination.vue`，内部固定 `layout="total, sizes, prev, pager, next, jumper"` + `:page-sizes="[10, 20, 50, 100]"`，props 只暴露 `v-model:page`/`v-model:pageSize`/`total`，emit 统一的 `change`（size 变更时自动把 page 重置为 1，这是目前 operation-logs:178 和 grants:613 各自手写的逻辑）。15 处调用点全部替换，AdminContentLayout 的 `#pagination` 插槽文档里注明只放这个组件。

<details><summary>核验记录</summary>

全仓 grep 'layout="' 返回 15 条 ElPagination，与描述的 4 种形态、页面清单、行号一一对应且零偏差：total,prev,pager,next 共 11 处（reviews:305、reports:337、teachers:345、sensitive-words:355、apps:919、consents:330、audit-events:329、token-probe-evidence:394、identity-review:475、student-verification:392、freshman-verification:377）；operation-logs:174 是 prev,pager,next,sizes,total；BlacklistTable:159 与 AdmissionSessionTable:254 是 total,prev,pager,next,sizes；grants:609-610 是唯一带 jumper 且带 :page-sizes 的。:page-sizes 全仓只有 operation-logs:175 和 grants:610 两处。那 11 个页面的 query.pageSize 默认写死 20（如 reports:37、reviews:37、identity-review:65、apps:69），layout 无 sizes 即不渲染每页条数下拉，「钉死在 20 条」属实。AdminContentLayout.vue:39-41 确实只有一个不约束内容的 #pagination 插槽。方案里提到的两处手写重置逻辑也对得上（operation-logs:178 @size-change="refreshPage(1)"、grants:613 @size-change="resetPageAndFetch"）。唯一落地注意点（不影响判定）：content-management-load-errors.test.ts:64、open-platform-load-errors.test.ts:88、verification-load-errors.test.ts:56 用源码文本断言钉死了 '@current-change="fetchData"'，替换成共享组件时这几个快照断言需同步改。

</details>

---

#### `P2` 审核队列没有批量选择，实名审核驳回还要穿两层弹窗，且同类审核动作各页深度不一

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/admin/apps/web-ele/src/views/users/identity-review/index.vue`
- `clients/admin/apps/web-ele/src/views/content/reports/index.vue`
- `clients/admin/apps/web-ele/src/views/users/student-verification/index.vue`

**现状**

全 admin 目录 grep `type="selection"` / `selection-change` / `batch` 命中数为 0，没有任何表格支持多选。identity-review 操作列（:452-466）只有一个「审核材料」按钮 → 打开 detail 对话框（:480，width min(920px,92vw)，还要额外发一次 fetchIdentityDetail 请求）→ 通过走 ElPopconfirm（:611-631）；驳回走 rejectDetail → openRejectDialog，在 detail 对话框还开着的情况下再叠一个 420px 的 ElDialog（:635-646），submitReject 成功后才在 :234-235 一起关掉两层。而 content/reports:281-325 和 users/student-verification:353-369 的同类审核动作是行内 ElPopconfirm 一键完成。

**问题**

1) 审核是管理员最高频操作，一次只能处理一条：驳回 = 开详情 → 等详情接口 → 点驳回 → 弹第二层 → 输理由 → 确定，4 次点击 + 2 层模态堆叠，弹窗遮弹窗时下面那层的证据图已经看不清，管理员边看材料边写理由的动作被打断。2) 同为「审核通过/驳回」，reports 和 student-verification 是行内一键，identity-review 是双层弹窗，管理员在页面间切换时肌肉记忆失效。3) 积压 200 条待审时没有批量通过/批量驳回，也没有「全选本页」。

**建议改法**

1) 保留并优先做真正的痛点：identity-review 删掉 rejectDialogVisible/rejectTarget 整套二层弹窗（:634-660 + 脚本 openRejectDialog/submitReject 相关分支），改为点「驳回」时在 detail 对话框 footer 上方内联展开 textarea + 「确认驳回」，管理员可边看证据图边写理由；student-verification 的 :397-426 同样内联进行内展开区，两页节奏就对齐了。2) 批量选择只加在低风险队列：content/reports（三个动作本就无需理由）和 student-verification/freshman-verification 的批量驳回（统一理由）。identity-review 不提供「批量通过」——若确需提速，只允许「批量驳回（统一理由）」，通过仍须逐条经过 detailCanApprove 证据闸门；如果一定要批量通过，必须先设计「逐条证据已浏览」的显式确认态，否则不要做。3) 批量提交按条汇总结果（成功 N / 失败 M 并列出 ID）用 ElAlert 展示，同时沿用 identity-review 已有的 reviewingActionsByUserId 逐条禁用，避免重复提交。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

1) identity-review 把驳回理由内联进 detail 对话框底部——点「驳回」时在 footer 上方展开一个 textarea + 「确认驳回」，不再开第二层 ElDialog（删掉 rejectDialogVisible 整套 state）。2) 给 identity-review / student-verification / freshman-verification / content-reports 的表格加 `<ElTableColumn type="selection" width="48">` + `@selection-change`，在 AdminContentLayout 的 #toolbar 里加一条选中态操作条：「已选 N 项 / 批量通过 / 批量驳回（统一理由）/ 取消选择」，批量驳回复用同一个理由输入。3) 批量提交后按条返回结果，用 ElMessage + 一条 ElAlert 汇总「成功 N 条，失败 M 条（列出 ID）」，不要只弹一个笼统失败。

</details>

<details><summary>核验记录</summary>

核心事实成立：全 admin views 目录 grep `type="selection"` / `selection-change` / `batch` 命中确为 0；identity-review/index.vue:446-467 操作列只有一个「审核材料」按钮，:479-483 detail 对话框 width=min(920px,92vw)，:634-637 在 detail 仍打开时叠第二层 420px ElDialog，submitReject（:232-236）成功后才一起关两层，确实是 4 次点击 + 双层模态。但两处描述失真、且方案有一项会造成合规倒退：(1) 「student-verification 是行内 ElPopconfirm 一键完成」只对通过成立——该页驳回同样走 index.vue:377 `openRejectDialog(row)` → :397-426 的 420px ElDialog，只是没有底层详情弹窗；真正「三个动作全行内」的只有 content/reports:280-327。所以「肌肉记忆失效」的落差比描述的小。(2) identity-review 的通过按钮受 :87-89 `detailCanApprove = detailHasRequiredEvidence && !detailHasFailedEvidence` 门控（模板 :622 也再判一次），即证件正反面/自拍必须真的加载成功才能通过——这是刻意的实名材料审核闸门；原方案 2) 的「批量通过」会绕过它，让管理员在没看过任何证据图的情况下批量放行实名认证，属于把可用性换成审核合规风险。另外该页无任何功能损坏，只是效率问题，P1 偏高。

</details>

---

#### `P2` 操作日志页零筛选条件，全站 15 个可筛选页面只有 1 个提供「重置」

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue`
- `clients/admin/apps/web-ele/src/views/authorization/grants/index.vue`

**现状**

operation-logs/index.vue:72 的 AdminContentLayout 完全没有 `#toolbar` 插槽——没有管理员筛选、没有动作类型筛选、没有资源类型筛选、没有时间范围，进页面就是倒序全量列表 + 分页。:136/:146/:156 的「变更前/变更后/请求」三列直接 `formatJSON(row.oldValue)` 塞原始 JSON 串，靠 `show-overflow-tooltip` 截断。另一边，`admin.common.reset`（'重置'）这个 key 在 locale 里是存在的，但全站只有 authorization/grants/index.vue:395-396 一处渲染了重置按钮（配 :182 的 resetFilters）；其余 14 个带筛选的页面只有「查询」，清筛选要手动逐个下拉选回「全部」。表格空结果时也没有任何页面做区分，一律落到 Element 默认的「暂无数据」。

**问题**

操作日志是出事后追责的唯一手段，却只能靠肉眼翻页找某个管理员在某天做了什么——量一上来就等于没有。JSON 原文挤在 220px 列里，要看清一次变更得逐个 hover。筛选页那边，管理员连点几个下拉后想回到全量视图，没有一键出口；筛出 0 条时看到的「暂无数据」也分不清是真没数据还是筛过头了。

**建议改法**

拆成两件事推进。(A) 操作日志筛选（真正的主项，全栈改动而非前端补插槽）：先在 server/api/paths/review-admin.yaml:350 的 getOperationLogs 增加 adminUsername/action/resourceType/startAt/endAt 查询参数，同步改 repository_operation_log.go:52 的 ListOperationLogs 签名与 SQL、service_admin.go:520 的调用，再重生成 OpenAPI 客户端并放开 clients/admin/apps/web-ele/src/api/admin/content.ts:152 的 params 类型；前端最后才补 #toolbar（管理员输入框 + 动作/资源 ElSelect + ElDatePicker daterange + 查询/重置）。改造前明确工作量是全栈，别按 UI 小改排期。(B) 重置按钮：模式已经存在于 3 处，不要以 grants 为唯一范本——直接把 BlacklistFilters.vue:93 / AdmissionSessionFilters.vue:69 的 emit('reset') 写法抽成 shared 的 AdminFilterActions.vue（内含查询+重置），推到其余 14 个 toolbar 页面；顺手把这两处写死的 '查询'/'重置' 换成 $t('admin.common.query')/$t('admin.common.reset')（两个 key 都已存在），与 i18n 那条发现合并处理。(C) 空态：先给 shared/admin-table/PersistentAdminTable.vue 增加一个透传到 ElTable 的 #empty 插槽（当前 :148 只有默认插槽），再谈各页面区分「无数据」与「筛过头了」。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

1) operation-logs 补 `#toolbar`：管理员（ElInput 搜用户名）+ 动作类型（ElSelect，选项从后端已有的 action 枚举来）+ 资源类型 + ElDatePicker 的 daterange，配「查询」「重置」。2) 「变更前/变更后」两列合并成一列「变更」，只显示被改动的字段名（`Object.keys(diff)` 摘要），点击行展开 ElTable 的 `type="expand"` 或开详情抽屉看格式化后的 JSON diff，别在列里塞原文。3) 把 grants 的「重置」按钮模式推到其余 14 个页面（就在「查询」右边加一个 `<ElButton @click="resetFilters">{{ $t('admin.common.reset') }}</ElButton>`）。4) 表格补 `#empty` 插槽：有筛选条件时显示「未找到匹配记录」+ 一个「清空筛选」按钮，无筛选时显示常规空态。

</details>

<details><summary>核验记录</summary>

前半真、后半假，且方案严重低估了范围。真的部分：content/operation-logs/index.vue:72 的 AdminContentLayout 确实只传了 title/total，全仓 17 处 '#toolbar' 里没有它，进页面就是倒序全量列表；:136/:146 两列确实用 formatJSON 直接塞 JSON 串靠 show-overflow-tooltip 截断（:156「请求」列其实是 ip+userAgent，不是 JSON，描述略有夹带）。假的部分（把已有机制说成缺失）：'全站只有 authorization/grants/index.vue:395-396 一处渲染了重置按钮' 不成立——users/member-blacklist/BlacklistFilters.vue:93 `<ElButton @click="emit('reset')">重置</ElButton>` 和 users/admission-sessions/AdmissionSessionFilters.vue:69 同样的重置按钮都存在，实际是 17 个 toolbar 页面里 3 个有重置、14 个没有，不是 15 选 1。方案不可行处：原方案把 operation-logs 补筛选写成纯前端活儿（'选项从后端已有的 action 枚举来'），但 server/api/paths/review-admin.yaml:350-360 的 getOperationLogs 只声明了 PageParam/PageSizeParam，server/internal/modules/course/review/repository_operation_log.go:52 的签名就是 ListOperationLogs(ctx, limit, offset)，后端根本没有任何过滤能力；另外第 4 条 '表格补 #empty 插槽' 也落不了地——shared/admin-table/PersistentAdminTable.vue:148 只把默认插槽透进 ElTable，调用点写 <template #empty> 会被丢弃，必须先改 PersistentAdminTable。

</details>

---

### UniApp X 跨端

共 9 条：P0 0 / P1 4 / P2 5

#### `P1` 6 个列表页没有 error 态：请求失败静默降级成「暂无数据」，且无重试入口

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/uniappx/src/pages/course/index.vue:33`
- `clients/uniappx/src/pages/course/index.vue:73`
- `clients/uniappx/src/pages/review/index.vue:161`
- `clients/uniappx/src/composables/usePagedList.ts:15`
- `clients/uniappx/src/pages/course/detail.vue:244`

**现状**

course/index.vue:33-38 的 `onError` 只做 `uni.showToast({icon:'none'})`；模板 :73-76 只有两个分支——`v-if="loading"` 显示「加载中」，`v-else-if="courses.length === 0"` 显示 `course.index.noResults`（暂无课程）。usePagedList 失败时 items 保持为空，于是 UI 落到「空」分支。user/reviews.vue:59、user/votes.vue:59、user/favorites.vue:59-60、user/notifications.vue:97-98、review/index.vue:161-162 完全同构。对照 course/detail.vue:244-249 与 teacher/profile.vue:67-72 —— 这两个详情页有 `loadError && !course` 分支 + `.retry-btn` 重试按钮。

**问题**

断网/后端 500 时，用户看到的是一句 1.5 秒就消失的 toast，然后停留在「暂无课程」「暂无评课」的静态页面上——语义完全错了：用户会以为库里真的没数据，而不是加载失败。而且没有任何重试入口，唯一的恢复手段是退出页面重进（因为 onShow 里 `if (courses.value.length > 0) return` 才会重新拉）。同一个 App 里详情页有重试、列表页没有，行为还不一致。

**建议改法**

保留原方案主体：usePagedList 增加 `loadError: Ref<unknown | null>`（refresh 开始清空、成功清空、失败写入；loadMore 失败单独用 moreError 以免整页翻成错误态），抽 src/components/ListState.vue 统一渲染 loading / error / empty 三态，error 分支带 role="alert" + 复用 course/detail.vue:246 同款 .retry-btn 调 refresh()，与 detail/teacher 页保持同一视觉与 data-testid 命名。6 个列表页把两分支换成 ListState。补充两点：(a) review/index.vue 有自维护的 loading/loadingMore/hasMore（:21-28、:34-87），迁到 usePagedList 时要一并保留它 onShow 无条件刷新的语义；(b) 描述与验收标准里应改写为「失败后语义错成空态、缺显式重试入口」，不要写成「静默失败/只能杀进程」，因为 toast 与 onShow 重拉（course/index 靠 length 判空、user/* 靠 30s STALE_MS 且失败不盖章）都已存在。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

在 usePagedList 里加 `loadError: Ref<unknown | null>`（refresh 成功清空、失败写入），并抽出 src/components/ListState.vue 统一渲染 loading / error(带重试按钮，调 refresh) / empty(带主行动，如「去发布评课」)。6 个列表页把 `v-if/v-else-if` 两分支换成 `<ListState :loading :error :empty="items.length===0" @retry="refresh">`。review/index.vue 迁到 usePagedList 后自动获得（见下条）。

</details>

<details><summary>核验记录</summary>

核心事实全部属实且行号精确：course/index.vue:32-37 的 onError 只 uni.showToast；模板 :73 `v-if="loading"` / :74 `v-else-if="courses.length === 0"` 只有两分支；review/index.vue:161-162、user/reviews.vue:59-60、user/votes.vue:59-60、user/favorites.vue:59-60、user/notifications.vue:97-98 同构；usePagedList.ts:15 的 onError 签名与 :52-58 catch 分支确实不写任何 error 状态，items 保持为空所以落到「空」分支；对照 course/detail.vue:244-249 与 teacher/profile.vue:67-72 确有 `loadError && !course` + .retry-btn，App 内行为不一致成立。方案（usePagedList 加 loadError + 抽 ListState）可行。但 P0 过高，两点支撑被代码削弱：(1) 并非「静默」，onError 确实弹了 toast；(2)「唯一恢复手段是退出重进（因为 onShow 里 courses.length > 0 才 return）」只对 course/index.vue:54 成立——user/reviews.vue:52、votes.vue:52、favorites.vue:52、notifications.vue:85 用的是 `Date.now() - lastLoadedAt < STALE_MS(30_000)`，而 lastLoadedAt 仅在 refresh 成功后才盖章（如 notifications.vue:44-47），失败后下一次 onShow 会立刻重拉；review/index.vue:141-143 更是每次 onShow 无条件 loadReviews()。即语义错误真实存在，但恢复成本远没到「必须杀进程」。降级为 P1。

</details>

---

#### `P1` review/index.vue 手写了一份分页状态机，与 usePagedList 双份实现

> 核验确认　|　工作量：S

**位置**

- `clients/uniappx/src/pages/review/index.vue:26`
- `clients/uniappx/src/composables/usePagedList.ts:34`

**现状**

5 个列表页（course/index、user/reviews、user/votes、user/favorites、user/notifications）用 composables/usePagedList.ts。review/index.vue:21-29 自己声明了 loading/loadingMore/page/total/hasMore/loadGeneration 六个 ref，:30-58 的 loadReviews 和 :60-88 的 loadMore 逐行复刻了 usePagedList.ts:41-96 的同一套逻辑：generation 竞态守卫、成功后才提交 page.value、失败不跳页、items 拼接。唯一差别是 `hasMore` 在这里是普通 ref、需要在 :47 和 :77 两处手动同步，而 composable 里是 computed。

**问题**

同一套非平凡的并发/竞态逻辑维护两份。上一条要给 usePagedList 加 loadError 时，review 广场页（App 的核心页面之一，挂在 tabBar 上）会被漏掉；反过来 usePagedList 的单测覆盖不到这份手写实现。手写的 hasMore 一旦有分支忘记同步，就会出现「还有数据但按钮消失」或「点了没反应」。

**建议改法**

review/index.vue 改用 `usePagedList({ pageSize: DEFAULT_PAGE_SIZE, fetchPage: (page, size) => unwrapListData(api.review.getLatestReviews({ page, pageSize: size, sort: sort.value })), onError })`，排序切换时调 refresh()（composable 的 generation 机制已覆盖切排序的竞态）。删除 :21-29 的六个 ref 和 :30-88 两个函数，约 -65 行。

<details><summary>核验记录</summary>

逐行核对属实。clients/uniappx/src/pages/review/index.vue:21-29 确实手写了 loading/loadingMore/page/total/hasMore 五个 ref 加 loadGeneration；:31-59 的 loadReviews 与 composables/usePagedList.ts:43-66 的 refresh 逻辑一一对应（++generation、loading=true 且 loadingMore=false、成功后才 page.value=1、generation 不匹配就丢弃响应、finally 里按 generation 复位 loading）；:61-90 的 loadMore 与 usePagedList.ts:68-93 同构（入口守卫 loading/loadingMore/!hasMore、requestGeneration=generation 不自增、nextPage 成功后才提交 page.value、items 拼接、total 覆盖）。差异也如描述：composable 里 hasMore 是 computed(usePagedList.ts:40)，这里是普通 ref 需在 :47 和 :78 两处手动同步。另有 5 个列表页在用该 composable（course/index.vue:21、user/reviews.vue:25、user/notifications.vue:27、user/favorites.vue:25、user/votes.vue:25），单测只覆盖 composables/usePagedList.test.ts，手写实现零覆盖。方案可行：按 course/index.vue:21-38 的现成写法即可（fetchPage 内闭包读 sort.value，changeSort 改调 refresh()，generation 机制覆盖切排序竞态）；原文那行伪代码漏了 await（unwrapListData 是同步函数，需 `async fetchPage(page,size){ const r = await api.review.getLatestReviews({...}); return unwrapListData(r) }`），属笔误不影响可落地性。review/index 是 pages.json:110 挂在 tabBar 上的核心页，维持 P1 合理。

</details>

---

#### `P1` 原生导航栏标题与 tabBar 文案硬编码中文，英文用户先看到中文再闪回

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/uniappx/src/pages.json:19`
- `clients/uniappx/src/i18n/index.ts:140`
- `clients/uniappx/src/i18n/index.ts:127`
- `clients/uniappx/src/pages/course/index.vue:53`

**现状**

pages.json 里 13 个页面的 navigationBarTitleText 全是中文字面量（"课程列表"、"评课广场"、"发布评课"、"我的收藏"…），tabBar 四个 text 也是「首页/课程/评课/我的」。运行时靠 i18n/index.ts:127 的 setPageTitle 在页面 onShow 里补丁式改写（course/index.vue:53、review/index.vue:142、user/index.vue:128 等 10 处）。i18n/index.ts:136-155 的 syncAppChrome 负责翻译 tabBar，但 :140 一行 `if (isH5Runtime()) return` 直接跳过 H5。

**问题**

en-US 用户每次进页面都会看到原生标题栏先渲染中文「课程列表」、下一帧才变成「Course List」的闪烁，这个闪烁在小程序端尤其明显（原生栏比 webview 先绘）。更硬的是 H5 构建（dist/build/h5 是实际产物之一）：syncAppChrome 直接 return，底部 tabBar 永远是「首页/课程/评课/我的」，英文用户全程看中文导航。i18n 做了两份 181 条完整词典，却在最外层的导航壳上漏了。

**建议改法**

(1) pages.json 12 个中文 navigationBarTitleText 统一改成 "StuHelper"（与 globalStyle:125 一致），标题全部交给 setPageTitle。(2) setPageTitle 可从 onShow 挪到 onLoad（全仓无运行时 setLocale 调用，语言只在 main.ts:7 bootstrap 时确定，不存在切换后需重设标题的场景）；course/detail.vue 和 teacher/profile.vue 里 onLoad+onShow 的重复调用去掉一处。(3) 修 H5 tabBar 必须同时做两件事：a) 把 syncAppChrome 的调用点从 createApp 阶段挪到首个 tabBar 页面激活之后（App.vue 的 onShow，或四个 tab 页 onShow 里调一次幂等的 syncAppChrome），否则 uni-h5 会以 'not TabBar page' 拒绝；b) 因为 uni.setTabBarItem 是 async API，改用 `runtime.setTabBarItem({ index, text, fail: (err) => { if (!shouldIgnoreTabBarSyncError(err.errMsg)) emitLocaleDiagnostic(...) } })` 或对返回值 .catch()，并同步修改 i18n/index.ts:14 的 UniRuntime 类型签名；现有 try/catch 对它无效。(4) 验证后再删 :140 的 isH5Runtime 早退；若 uni-h5 tabBar store 仍不生效，退回自绘 tabbar 方案。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

pages.json 全部 navigationBarTitleText 改为中性的 "StuHelper"（或空串），标题完全交给 setPageTitle；把 setPageTitle 从 onShow 移到 onLoad（更早一帧，且不必每次切回都重设）。H5 端删掉 i18n/index.ts:140 的早退，改为对 setTabBarItem 的失败做 try/catch（已有 shouldIgnoreTabBarSyncError）；若 uni H5 的 setTabBarItem 确实不生效，则在 App.vue 挂一个自绘 tabbar 组件（同时能解决上一条 tabBar 配色不一致）。

</details>

<details><summary>核验记录</summary>

问题真实且比描述更严重，但修复方案的关键一步经证实不成立。事实核对：pages.json 里 12/13 个页面的 navigationBarTitleText 是中文字面量（:19 课程列表、:31 评课广场、:37 发布评课、:73 我的收藏…），只有 home（:12）是 "StuHelper"，且 home/login/callback 三页是 navigationStyle:"custom" 不渲染原生栏，因此受闪烁影响的是 10 个页面而非 13；tabBar 四项文案（pages.json:99/105/111/117）确为中文；i18n/index.ts:127 的 setPageTitle 在 10 个页面的 onShow/onLoad 里补写（course/index.vue:53、review/index.vue:142、user/index.vue:128 等 12 处调用）；i18n/index.ts:140 `if (isH5Runtime()) return` 确实让 syncAppChrome 在 H5 完全跳过 tabBar 翻译。而 package.json 只有 dev:h5/build:h5 两个构建脚本，H5 是唯一产物，所以"英文用户全程看中文导航"是常态（原文"小程序端尤其明显"反而无对应构建）。方案缺陷（决定性）：删掉 :140 早退 + 依赖 try/catch 修不好 H5。node_modules/@dcloudio/uni-h5 里 setTabBarItem 是 defineAsyncApi(:23395) 经 promisify(:2950) 包装，无回调时返回 Promise，失败走 reject 而不是 throw，现有 :150-155 的 try/catch 根本捕获不到；更关键的是 setTabBar(:23330-23341) 要求 getCurrentBasePages() 非空且当前页 meta.isTabBar，而 syncAppChrome 只从 bootstrapLocale 调用、bootstrapLocale 在 main.ts:7 的 createApp() 里、早于任何页面创建，必然 reject 'not TabBar page'，结果是 tabBar 依旧中文外加 4 条未处理的 Promise rejection。（好消息：playwright.config.ts locale 固定 zh-CN，改动不会踩 tests/e2e/surface.spec.ts:750 那条 '首页' 断言。）

</details>

---

#### `P1` 没有设计 token：185 处硬编码色值 / 21 种颜色，且与 web 主站不是同一套视觉

> 核验确认　|　工作量：L

**位置**

- `clients/uniappx/src/pages.json:92`
- `clients/uniappx/src/App.vue:112`
- `clients/uniappx/src/pages/home/index.vue:210`
- `clients/web/src/styles/tailwind.css:18`

**现状**

13 个页面 + App.vue 里有 185 处十六进制色值、去重后 21 种，没有任何 CSS 变量或 token 文件（uniappx 下无 styles/ 目录）。主色 #4f46e5 出现 29 次、#64748b 28 次、#0f172a 23 次，全是逐页复制的 Tailwind slate/indigo 值。对照 web：tailwind.css:18 `--color-primary: #3f5ccb`、:23 `--color-accent: #e87aac`（粉）、:39 `--color-bg-base: #ece8f4`（淡紫）、:79-84 圆角阶梯 4/6/10/14/18/24px、:231+ 一整套 dark 变量。uniappx 圆角是 22/24/28/32rpx 随手写的，且完全没有暗色模式。App 内部还自相矛盾：pages.json:93 tabBar selectedColor `#6366F1` vs 页面主色 `#4f46e5`；pages.json:92 tabBar color `#8B8B8B` vs 图标 PNG 实际灰色 `#94a3b8`；pages.json:127 globalStyle backgroundColor `#F8F9FA` vs App.vue:112 `page { background-color: #f8fafc }`。

**问题**

两端并排放会被认成两个产品：web 是淡紫底 + 蓝紫主色 + 粉色强调，uniappx 是纯 slate 灰底 + 靛蓝，圆角字号也对不上。App 内部 tab 栏选中态（靛蓝 500）比页面按钮（靛蓝 600）浅一档、tab 文字灰和图标灰是两个灰、页面背景在 globalStyle 和 page 规则之间差 1 个色值——滚动到顶部回弹时能看到两块底色。185 处散落的 hex 意味着任何一次品牌调色都要改 13 个文件，暗色模式基本无法追加。

**建议改法**

新增 clients/uniappx/src/styles/tokens.css，用与 web @theme 同名的变量（--color-primary / --color-accent / --color-bg-base / --color-text-secondary / --radius-lg 等）承接 web 的实际取值，在 main.ts 里 import；App.vue 的 `page` 规则改用变量。用 sed 批量把 21 个 hex 映射到变量（#4f46e5→--color-primary、#64748b→--color-text-secondary、#0f172a→--color-text-primary、#f8fafc→--color-bg-base…）。pages.json 的 tabBar color/selectedColor/backgroundColor 和 globalStyle backgroundColor 对齐同一组值，并按新主色重新导出 8 张 tabbar PNG（现在是 64x64 单色图，重上色成本近乎为零）。

<details><summary>核验记录</summary>

逐项实测全部对上。clients/uniappx/src 下 grep -roh '#[0-9a-fA-F]{3,8}' --include=*.vue 计数正好 185，去重正好 21；频次 #4f46e5=29、#64748b=28、#0f172a=23 与发现完全一致；grep 'var(--' --include=*.vue 命中 0，src/styles/ 目录不存在。web 侧 clients/web/src/styles/tailwind.css:18 `--color-primary: #3f5ccb`、:23 `--color-accent: #e87aac`、:39 `--color-bg-base: #ece8f4`、:79-84 `--radius-xs..2xl` = 4/6/10/14/18/24px、:229-231 起 `[data-theme="dark"]` 一整套暗色变量，行号逐行精确。uniappx 圆角实测为 16/18/20/22/24/28/32/54rpx 随手写。pages.json:92 `"color": "#8B8B8B"`、:93 `"selectedColor": "#6366F1"`、:127 `"backgroundColor": "#F8F9FA"`，App.vue 的 page 规则为 #f8fafc（在 :115-116，发现写 :112 差 3 行，可接受）。连「图标 PNG 实际灰 #94a3b8 / 64x64 单色」都核实了：用 PIL 读 static/tabbar/*.png，home/course/review/user.png 唯一不透明色恰为 (148,163,184)=#94a3b8，home-active.png 为 (99,102,241)=#6366f1，尺寸均 64x64。方案在 H5（唯一正式构建目标）可行，CSS 变量可用。仅一处夸大：#F8F9FA vs #f8fafc 只差 B 通道 2/255 且 globalStyle.backgroundColor 在 H5 上并不生效，「回弹能看到两块底色」站不住；但这只是佐证细节，不影响主结论与 P1 定级。

</details>

---

#### `P2` 3563 行页面代码只有 1 个组件：.state-card 被定义 14 次，评课卡片整块复制

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：L

**位置**

- `clients/uniappx/src/pages/user/favorites.vue:94`
- `clients/uniappx/src/pages/user/index.vue:212`
- `clients/uniappx/src/pages/review/index.vue:164`
- `clients/uniappx/src/pages/course/detail.vue:315`
- `clients/uniappx/src/components/A11yButton.vue:1`

**现状**

components/ 下只有 A11yButton.vue（45 行）。跨页面重复的选择器定义次数：`.state-card` 14 处（course/detail.vue:371,380、course/index.vue:147、review/index.vue:229、review/post.vue:288,296、teacher/profile.vue:125,134、user/favorites.vue:94、user/index.vue:212 和 :276（同一文件里定义了两次）、user/notifications.vue:151、user/reviews.vue:86、user/votes.vue:86）、`.hero-card` 7 处、`.primary-btn` 6 处（login.vue 里同样重复定义两次）、`.load-more` 6 处（course/index.vue 却叫 `.more-btn`）。评课卡片 markup 在 review/index.vue:164-176 与 course/detail.vue:315-323 一字不差地复制（.review-card/.review-score/.review-teacher/.review-content/.review-meta），但截断长度不同：160 vs 180；评分渲染有 3 套写法——`averageRating(review.ratings)`(review/index.vue:170)、`t('common.scorePrefix',{value: averageRating(...)})`(user/reviews.vue:75、user/votes.vue:75)、手写 `teacher.avgRating ? teacher.avgRating.toFixed(1) : '--'`(course/detail.vue:306、teacher/profile.vue:80,111)。

**问题**

同一个空状态卡片有 14 份样式副本，其中 user/favorites/reviews/votes 三份是压成一行的 `.state-card { margin: 24rpx; padding: 40rpx; background: #fff; ... }`，而 user/index.vue 那份 padding 是 40rpx 但圆角来自另一条合并规则——实际渲染出来的空态在不同页面圆角和阴影不一致。同一条评课在广场页截断 160 字、在课程详情页截断 180 字，用户会觉得内容莫名其妙多了一截。评分在有的页面显示「4.5」有的显示「评分 4.5」有的显示「--」，看不出是同一个指标。任何一次卡片改版都要改 14 个地方，必然漏。

**建议改法**

(1) 抽 StateCard.vue（loading/empty/error+retry 三态）替换 10 个页面的 .state-card 分支，统一是否带 box-shadow；抽 LoadMoreButton.vue 统一 .load-more/.more-btn；抽 ScoreText.vue 或在 utils/format.ts 增加 formatRatingValue(value|ratings, {prefix?}) 统一 '--' 兜底与是否带"综合"前缀（course/detail.vue:306、teacher/profile.vue:80,111 改为调用它）。(2) 截断长度提取为 config 常量（如 REVIEW_EXCERPT_LENGTH=180）供 review/index、course/detail、user/reviews、user/votes 共用，先统一数值再谈组件化。(3) ReviewCard 只在 user/reviews.vue 与 user/votes.vue 之间抽（这两处 markup 与样式确实逐行相同）；review/index.vue 与 course/detail.vue 的卡片头部/底部差异较大，若要合并需用具名插槽（#head/#footer）承载投票行与回复区，不能按"一字不差"直接替换。CourseListRow 只在 home/index.vue:186-198 与 teacher/profile.vue:100-112 之间抽（结构一致），course/index.vue:78-91 的卡片带课程代码/评课数徽章/学分，不属于同一形态。(4) 删除 CSS 冗余时用"合并"而非"删除"：auth/login.vue:120-134 两条 .primary-btn 合并成一条；user/index.vue:276 不是重复规则，禁止删除。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

抽 5 个组件放 src/components/：StateCard.vue（loading/empty/error+retry 三态，替换 14 处）、ReviewCard.vue（props: review, truncateLength=160，替换 review/index + course/detail + user/reviews + user/votes 四处）、CourseListRow.vue（course/index.vue:78-93 与 home/index.vue:186-198 的热门课程行、teacher/profile.vue:107-113 的授课列表共用）、RatingValue.vue（统一 `--` 兜底与是否带前缀）、LoadMoreButton.vue（统一类名与 loading 文案）。同时删掉 user/index.vue:276 和 auth/login.vue 里重复的第二份 .state-card / .primary-btn 规则。预计页面代码可减 700-900 行。

</details>

<details><summary>核验记录</summary>

核心事实成立但多处描述被夸大，且方案含破坏性步骤。成立部分：clients/uniappx/src/components/ 下确实只有 A11yButton.vue（45 行）；grep 到 .state-card 选择器 14 次（course/detail.vue:371,380；course/index.vue:147；review/index.vue:229；review/post.vue:288,296；teacher/profile.vue:125,134；user/favorites.vue:94；user/index.vue:212,276；user/notifications.vue:151；user/reviews.vue:86；user/votes.vue:86），.load-more 在 5 个页面各写一份而 course/index.vue:93/143 叫 .more-btn，截断 160（review/index.vue:173）vs 180（course/detail.vue:321、user/reviews.vue:74、user/votes.vue:74）也属实，评分渲染确有 3 套（review/index.vue:170、user/reviews.vue:75 的 t('common.scorePrefix')="综合 {value}"、course/detail.vue:306 与 teacher/profile.vue:80,111 的手写 '--' 兜底）。夸大/错误部分：(a) 14 次里有 4 次是同文件内的"基础组选择器 + 专属规则"（course/detail.vue:371 是 .state-card,.section-card,.hero-card 合并规则；teacher/profile.vue:125、review/post.vue:288、user/index.vue:210 同理），实际只有 10 个文件各一份，不是 14 份副本；(b) "圆角不一致"不成立——10 份全部 border-radius:24rpx、margin:24rpx、padding:40rpx、color:#64748b，唯一差异是 box-shadow（course/detail、teacher/profile、review/post、user/index 有，其余 6 个没有）；(c) "review/index.vue:164-176 与 course/detail.vue:315-323 一字不差地复制"不成立——前者 review-head 是包着 course-name 的 A11yButton（:165-171），后者是纯 view 只有 title+score（:316-319），review-meta 一个是 term+日期（:174-177）另一个是点赞/回复/学期（:322-333），只有 review-teacher/review-content 两行和类名相同；(d) 页面总行数实测 3392 行不是 3563。方案缺陷："删掉 user/index.vue:276 和 auth/login.vue 里重复的第二份规则"会造成真实样式回归——user/index.vue:276-280 是唯一提供 padding/text-align/color 的规则（:212 那条只给 margin/圆角/阴影），删掉后加载态卡片没内边距不居中；auth/login.vue:131-134 提供 background:#4f46e5;color:#fff，删掉后登录主按钮变默认灰白。综合看这是纯可维护性问题，用户可见差异只有阴影和 160/180 截断，定 P1 偏高。

</details>

---

#### `P2` 全端零安全区/状态栏适配，3 个自定义导航页内容压在刘海下

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/uniappx/src/pages.json:13`
- `clients/uniappx/src/pages/home/index.vue:205`
- `clients/uniappx/src/pages/auth/login.vue:80`
- `clients/uniappx/src/App.vue:112`

**现状**

pages.json:13/80/87 三个页面（home/index、auth/login、auth/callback）声明 navigationStyle: "custom"，即完全去掉原生导航栏、自己画头部。但全仓库 14 个页面 + App.vue 对 statusBar / safe-area / getWindowInfo / env(safe-area-inset-*) 的 grep 结果是 0 命中。home/index.vue:205 `.home-page { min-height: 100vh }`，第一个内容块 `.hero-card`(:210) 只有 `margin: 24rpx`（≈12px）；login.vue:80 `.login-card { margin-top: 80rpx }`（≈40px）。四个 tab 页的 scroll-view 底部也没有任何 padding-bottom。

**问题**

刘海屏状态栏 44-47pt，home 页 hero 卡片顶部 12px 会被状态栏整个盖住——用户看到的是标题被时间/信号图标压住；login 卡片 40px 也压边。底部没有 safe-area-inset-bottom，iPhone home indicator 横条会盖住 tab 页 scroll-view 的最后一行内容和「加载更多」按钮，用户点不到最后一项。这是纯原生端才暴露的问题，H5 预览看不出来，所以一直没被发现。

**建议改法**

按「为未来原生交付预埋」的定位处理，不要当作线上缺陷抢修。1) 仍然新增 src/components/PageShell.vue 收口页面骨架：自定义导航页 `padding-top: var(--status-bar-height)`（H5 恒 0px，app-plus 下为真实值），底部统一 `padding-bottom: calc(24rpx + constant(safe-area-inset-bottom)); padding-bottom: calc(24rpx + env(safe-area-inset-bottom))`；home/login/callback 三页删掉写死的 margin-top: 80rpx 改由 PageShell 提供。2) 关键补充：若希望这层适配在 H5/PWA 也真正生效，必须同时把 clients/uniappx/index.html:5 的 viewport 改成 `width=device-width, initial-scale=1.0, viewport-fit=cover`，否则 env(safe-area-inset-*) 恒为 0，改动只是死代码——原方案漏了这一步。3) 把描述改成「当前 H5 目标不受影响，属于原生/小程序目标启用前的阻塞前置项」，并挂到 docs/guides/native-auth-qa-checklist.md 同级的原生上线前置清单里；同时删掉「四个 tab 页底部没有任何 padding-bottom」（.list-wrap 实有 24rpx）与「14 个页面」（实为 13 个）这两处不准确表述。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

新增 src/components/PageShell.vue 包住每个页面的 scroll-view：自定义导航页用 uni 内置 CSS 变量 `padding-top: var(--status-bar-height)`（uni-app 原生注入，H5 下为 0），非自定义页不加；所有页面统一 `padding-bottom: calc(24rpx + constant(safe-area-inset-bottom)); padding-bottom: calc(24rpx + env(safe-area-inset-bottom))`。home/login/callback 三页把 hero/card 的顶部间距改为 PageShell 提供，删掉写死的 margin-top: 80rpx。

</details>

<details><summary>核验记录</summary>

「零适配」这个事实成立：pages.json:13/:80/:87 三处 navigationStyle: "custom"（home/index、auth/login、auth/callback）；clients/uniappx/src 全量 grep safe-area|safeArea|statusBar|status-bar|getWindowInfo|env(safe 命中 0（只有 i18n/index.ts:79 的 getSystemInfoSync 读语言）；home/index.vue:205 `.home-page{min-height:100vh}`、:210 `.hero-card{margin:24rpx}`、auth/login.vue:80 `.login-card{margin-top:80rpx}` 均属实（App.vue 的 page 规则在 :115-116 而非 :112，差几行可接受）。但 P0 不成立，因为这个失效面在当前所有可交付目标上都不会发生：clients/uniappx/package.json 只有 dev:h5 / build:h5，依赖里只有 @dcloudio/uni-h5，无任何 app-plus/mp 编译器；scripts/check-uniappx-platform-contract.mjs 是 CI 门禁（.github/workflows/ci.yml:161-162），显式要求「H5 supported，mp-weixin 不得声明」；clients/uniappx/README.md 的「支持边界」写明「H5 是当前唯一有正式构建、产物契约和浏览器回归的 UniAppX 目标」，原生只被称作「原生实验代码」。而在 H5 上，clients/uniappx/index.html:5 的 viewport 是 `width=device-width, initial-scale=1.0`，没有 viewport-fit=cover，iOS Safari 不会把内容画到刘海/home indicator 下，env(safe-area-inset-*) 恒为 0，所以「标题被时间信号图标压住」在今天任何被构建的产物里都复现不出来。另有两处事实偏差：页面数是 13 不是 14；「四个 tab 页 scroll-view 底部没有任何 padding-bottom」不准确——course/index.vue:157 与 review/index.vue:239 的 .list-wrap 都是 `padding: 0 24rpx 24rpx`，course/index.vue:145 .more-btn 还有 `margin: 12rpx 24rpx 36rpx`，只是不感知安全区。方案本身技术上没错（uni-h5.es.js 的 initCssVar 确实注入 `--status-bar-height`，H5 下为 0px），但在 H5 上是纯 no-op。综上：真实的潜在债，非当下 P0，降为 P2。

</details>

---

#### `P2` 移动端零下拉刷新、零触底加载，翻页全靠点「加载更多」按钮

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/uniappx/src/pages.json:8`
- `clients/uniappx/src/pages/course/index.vue:93`
- `clients/uniappx/src/pages/review/index.vue:188`
- `clients/uniappx/src/pages/user/notifications.vue:125`

**现状**

pages.json 的 13 个页面 style 里没有任何一个声明 `enablePullDownRefresh`；全仓库 grep `onPullDownRefresh` / `onReachBottom` / `@scrolltolower` 命中数为 0。6 个列表页（course/index.vue:93、user/reviews.vue:77、user/votes.vue:77、user/favorites.vue:85、user/notifications.vue:125、review/index.vue:188）都在列表末尾放一个 A11yButton「加载更多」，文案在 loadingMore 时变成「加载中」。刷新数据的唯一路径是 onShow 里的 `if (items.length > 0) return` 缓存判断。

**问题**

下拉刷新和触底加载是移动端的肌肉记忆，用户进消息通知页第一反应就是下拉——什么都不会发生，会以为页面卡了。看完一屏还要精准点到底部一个按钮才能翻页，翻 5 页要点 5 次。更糟的是通知/我的评课这类会变化的数据完全没有主动刷新手段：列表非空时 onShow 直接 return，用户在别处发了评课回到「我的评课」看不到新数据，只能杀进程。

**建议改法**

不要走页面级 enablePullDownRefresh/onPullDownRefresh。这些页面根节点是 scroll-view，应改用 scroll-view 自带刷新器：`<scroll-view scroll-y refresher-enabled :refresher-triggered="refreshing" @refresherrefresh="onRefresh" @scrolltolower="loadMore" :lower-threshold="120">`，onRefresh 里 `refreshing = true; refresh().finally(() => refreshing = false)`。运行时已支持：uni-h5.es.js 有 refresherEnabled 处理，uni.d27cfdb1.css 里已内置 uni-scroll-view-refresher / -refresh-inner / -refresh__spinner 样式，无需改 pages.json。触底加载用 @scrolltolower（scroll-view 自身滚动，事件可正常触发），并把 usePagedList.loadMore 的 `if (loading || loadingMore || !hasMore) return` 当作天然防抖。「加载更多」按钮保留但降级为：loadingMore 时显示不可点的「加载中…」，失败时显示可点「重试」，以保留键盘/读屏可达路径。三件事封进 ListState + usePagedListPage()。同时把发现描述里「通知/我的评课完全没有主动刷新手段、只能杀进程」删掉，改为「只有 30s STALE_MS 被动刷新，缺用户可主动触发的下拉手势」。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

pages.json 给 6 个列表页加 `"enablePullDownRefresh": true`；页面里 `onPullDownRefresh(() => refresh().finally(() => uni.stopPullDownRefresh()))`。scroll-view 加 `@scrolltolower="loadMore" :lower-threshold="120"`，把「加载更多」按钮降级为：hasMore 且 loadingMore 时显示「加载中…」文本，加载失败时才显示可点的「重试」按钮（同时保留给键盘/读屏用户的可达路径）。这三件事应该封在上一条的 ListState + 一个 usePagedListPage() 页面级 composable 里，六个页面各改 3 行。

</details>

<details><summary>核验记录</summary>

现状事实成立：clients/uniappx/src 下 grep onPullDownRefresh|onReachBottom|scrolltolower|enablePullDownRefresh|refresher 命中数为 0；pages.json 13 个页面 style 均无 enablePullDownRefresh；6 个「加载更多」按钮行号逐一对上（course/index.vue:93、user/reviews.vue:77、user/votes.vue:77、user/favorites.vue:85、user/notifications.vue:125、review/index.vue:188）。但两处关键内容需要修正。其一，「刷新数据的唯一路径是 onShow 里的 if (items.length > 0) return」只适用于 course/index.vue:54；user/reviews.vue:52、votes.vue:52、favorites.vue:52、notifications.vue:85 用的是 `Date.now() - lastLoadedAt < STALE_MS`（各文件 :17/:19 定义 STALE_MS = 30_000），review/index.vue:141-143 每次 onShow 无条件 loadReviews()。所以「通知/我的评课完全没有主动刷新手段、只能杀进程」是错的——离开 30 秒再回来即自动刷新。其二，原方案的机制在本代码库会造成回归：6 个列表页根节点都是 <scroll-view scroll-y>，而 uni-h5 内置样式 `uni-scroll-view{position:relative;width:100%;height:100%;max-height:inherit}`（见 dist/build/h5/assets/uni.d27cfdb1.css）让 scroll-view 自己持有滚动，document 永远不滚；node_modules/@dcloudio/uni-h5/dist/uni-h5.es.js:27519-27535 的 usePageRefresh.onTouchmove 只用 `(document.documentElement.scrollTop || document.body.scrollTop) !== 0` 作为放行条件，这里恒为 0，于是列表滚到任意位置向下划都会命中 ev.preventDefault() 触发下拉刷新，用户将无法向上回滚。故 enablePullDownRefresh + onPullDownRefresh 是错的机制。综合「自动刷新其实已存在 + 剩余缺口是手势习惯、且已有可用且对键盘/读屏友好的按钮路径」，降级为 P2。

</details>

---

#### `P2` 能力边界宣称与实际不符：个人中心是只读死路，手机绑定不可点

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：M

**位置**

- `clients/uniappx/src/i18n/zh-CN.ts:41`
- `clients/uniappx/src/pages/user/index.vue:181`
- `clients/uniappx/src/pages/user/index.vue:39`
- `clients/uniappx/src/config/featureSurface.ts:11`

**现状**

featureSurface.ts:11 的 getUniappxAppNotice 取 i18n/zh-CN.ts:41 的文案「uniappx 已接入共享 OpenAPI 契约，课程、评课、登录和个人中心核心能力均可用。」，渲染在 home/index.vue:136 的 hero-caption。实际情况：user/index.vue:181-187 手机号那一行是普通 `<view>`（相邻的实名/学籍两行是 A11yButton），只显示「已绑定/未绑定」，没有任何点击行为；user/index.vue:39-70 的 goVerify 把实名和学籍认证整个甩到系统浏览器（`plus.runtime.openURL`），VITE_WEB_URL 为空时只弹一句 toast「请在网页端完成」。对照 web 的 router：uniappx 完全没有 /search、没有 /resources* 五条资源共享路由、没有 /user/phone-binding、/user/qq-binding、/user/academic-info、/user/authorized-apps、/account/*，也没有 404 兜底页，更没有 /admission/freshman/camera/:token —— 那是 web 专门为手机拍照做的移动端流程。

**问题**

首页那句话让用户以为个人中心该有的都有，进去发现「未绑定手机号」是个死字：不可点、没有说明、没有去处；想做实名认证会被踢出 App 到系统浏览器，在浏览器里还得重新登录一次。最反直觉的是 admission 的新生拍照流程——web 端设计成扫码跳手机浏览器，而真正有相机权限、最适合承载它的原生 App 反而一个页面都没有。

**建议改法**

(1) 改 i18n/zh-CN.ts:41 与 en-US.ts:41 的 feature.notice 为诚实边界，如"课程、评课、登录与个人中心浏览已可用；实名认证、手机绑定等资料操作需在网页端完成"，并同步 config/__tests__/featureSurface.test.ts 的断言。(2) user/index.vue:182-187 改成 A11yButton，未绑定时补 action-hint（新增 i18n key，如 user.index.bindPhoneOnWeb="点击前往网页端绑定"），点击复用网页跳转逻辑指向 /user/phone-binding，与上面两行交互一致。(3) 把 goVerify 抽成 src/utils/web-handoff.ts 的 openOnWeb(path)，统一处理 VITE_WEB_URL 为空的兜底、H5 的 location.href 与原生的 showModal+openURL；跳转 URL 只允许携带 ?redirect=<path>（回跳用），严禁附带 accessToken/refreshToken/sessionID——二次登录问题应通过 web 端与 App 共用同一 SSO 会话域或走 /login?redirect= 解决。(4) 路线图上把 /admission/freshman/camera 列为原生端优先项（manifest.json 已声明 app-plus 与 stuhelper scheme，具备承载条件）。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

(1) 改文案为诚实的边界声明，如「课程、评课、登录已可用；实名认证与资料绑定需在网页端完成」；(2) user/index.vue:181 手机行改成 A11yButton，复用 goVerify 的网页跳转逻辑（path 换成 /user/phone-binding），未绑定时补一句 action-hint「点击前往网页端绑定」，与实名/学籍行的交互保持一致；(3) 把 goVerify 抽成 openOnWeb(path) 放 src/utils/web-handoff.ts，跳转 URL 带上当前会话 token 或 redirect 参数避免二次登录；(4) 路线图上把 /admission/freshman/camera 列为原生端第一优先，它是唯一一个原生比 web 更合适的能力。

</details>

<details><summary>核验记录</summary>

手机行死路属实，但标题的"只读死路"以整体概括过度，且方案第 (3) 条与项目既有安全设计冲突。事实核对：config/featureSurface.ts:10-12 取 i18n/zh-CN.ts:41 的"…课程、评课、登录和个人中心核心能力均可用"（en-US.ts:41 同义），渲染于 home/index.vue:136；user/index.vue:182-187 的手机号那行确是普通 <view class="summary-item">，只显示 phoneBound ? 已绑定 : 未绑定，无 @tap 无 action-hint，而相邻两行(:161-181)是可点的 A11yButton；goVerify(:39-72) 确实把实名/学籍甩到网页端；web 端对照路由全部核实存在：/search(router/index.ts:361)、/resources 五条(:281,289,297,305,313)、/user/phone-binding(:452)、/user/qq-binding(:463)、/user/academic-info(:474)、/user/authorized-apps(:417)、/account/profile+security(:147,158)、404 兜底(:496)、/admission/freshman/camera/:token(:206)，uniappx pages.json 13 个页面全无对应。需要修正之处：(a) 个人中心并非"只读死路"——user/index.vue:191-200 的常用功能四项（featureSurface.ts:37-48 我的评课/投票/收藏/通知）、登出(:151-153)、以及实名/学籍两行都可用，真正的死字只有手机号一行；(b) goVerify 在 #ifdef H5 分支(:51-53)走的是 window.location.href，而 package.json 只有 build:h5，实际产物不会走 plus.runtime.openURL(:64)，"被踢出 App 到系统浏览器"描述的是尚未构建的原生分支（原生分支还有 uni.showModal 二次确认:55-70，并非无提示直跳）。方案缺陷：第 (3) 条"跳转 URL 带上当前会话 token"与本仓库的 token 处置方式直接冲突——api/native-session.ts:2-3,36-50 明确把原生 token 放进 secureStorage 桥接层并清理历史的 uni.setStorageSync 副本，把 access token 放进 URL 会经由浏览器历史/Referer/日志泄露，不能采纳。

</details>

---

#### `P2` 首页快捷入口与 tabBar 三个入口完全重合，是纯冗余导航

> **核验调整** —— 严重度或方案已被修正，下方「建议改法」为修正后版本　|　工作量：S

**位置**

- `clients/uniappx/src/config/featureSurface.ts:19`
- `clients/uniappx/src/pages/home/index.vue:155`
- `clients/uniappx/src/pages/home/index.vue:110`

**现状**

featureSurface.ts:19-38 的 getHomeFeatures 返回三个入口：/pages/course/index、/pages/review/index、/pages/user/index —— 正好就是 pages.json:98-118 里 tabBar 的第 2/3/4 项。home/index.vue:155-169 把它们渲染成一个占满一屏宽的 3 宫格「快捷入口」，home/index.vue:110-116 的 go() 还得专门写一段 tabPages 白名单来把 navigateTo 改成 switchTab。

**问题**

首页最显眼的一块区域，三个大卡片指向的位置和屏幕底部一直可见的 tab 图标一模一样，点进去 tab 高亮还会跳变。用户第一次看会以为是不同的东西，点开发现就是「课程」，信息价值为零，还把真正有价值的「热门课程」列表挤到了第一屏之下。真正缺入口的功能（发布评课 /pages/review/post、我的收藏、课程搜索）反而没有任何首页入口。

**建议改法**

(1) getHomeFeatures 换成真正缺入口且可独立打开的目的地："我的收藏 → /pages/user/favorites"、"消息通知 → /pages/user/notifications"（红点需先在 home 拉未读数，可复用 api.notification 的未读统计，无现成数据就先不做红点）；"发布评课"不要直接指向 /pages/review/post，改为指向 /pages/course/index（带 ?intent=post 之类的标记，由课程列表选课后再进 post），或干脆换成"课程搜索 → /pages/course/index"。(2) go() 必须保留 tabPages→switchTab 分支，因为 home/index.vue:176 的"查看全部"仍指向 tab 页；可以只把 :105 那层多余的 startsWith 前置判断简化掉，白名单本身要留。(3) 若采纳"整块删掉快捷入口区"的激进版，同时要删 home/index.vue:119-124 的 shortcutKey 与 featureSurface.ts:14-35，并同步删除 tests/e2e/surface.spec.ts:756-786 三条 shortcut 用例和 config/__tests__/featureSurface.test.ts 中对应断言。

<details><summary>评审员原方案（已被核验修正，仅存档）</summary>

getHomeFeatures 换成非 tab 目的地：`发布评课 → /pages/review/post`、`我的收藏 → /pages/user/favorites`、`消息通知 → /pages/user/notifications`（带未读红点）。相应地 home/index.vue:110-116 的 go() 可以删掉 tabPages 白名单分支，只保留 navigateTo。或者更激进：整块删掉「快捷入口」区，把热门课程提到 hero 卡片正下方，首屏直接给内容。

</details>

<details><summary>核验记录</summary>

现状描述准确，但两条落地建议都会造成回归。事实核对：config/featureSurface.ts:14-35 的 getHomeFeatures 返回 /pages/course/index、/pages/review/index、/pages/user/index，正是 pages.json:104/110/116 的 tabBar 第 2/3/4 项；home/index.vue:153-168 把它们渲染成 grid-template-columns:repeat(3,1fr)(:291-296) 的三宫格；go()(:104-113) 里确有 tabPages 白名单转 switchTab；热门课程列表被排在 hero(:133-151)+快捷入口区之后，首屏挤压属实；/pages/review/post、/pages/user/favorites 在首页确无入口。方案缺陷一："go() 可以删掉 tabPages 白名单分支，只保留 navigateTo" 会直接打断 home/index.vue:173-179 的"查看全部"按钮——它调用 go('/pages/review/index')，而 uni-h5 的 normalizeUrl(node_modules/@dcloudio/uni-h5/dist/uni-h5.es.js:6025-6028) 对 navigateTo 到 isTabBar 页面直接返回 'can not navigateTo a tabbar page'，tests/e2e/surface.spec.ts:788-797 的 'home hot courses view-all opens the review square tab' 会挂。方案缺陷二：把"发布评课"指向 /pages/review/post 是死路——review/post.vue:15/51-52 依赖 onLoad 传入的 courseID，无 courseID 时 loadPage 直接 return，模板 :186 渲染 "缺少课程上下文，无法发评课"，用户点首页大卡片会撞到错误态。

</details>

---

---

## 7. 建议的修复批次

| 批次 | 内容 | 量级 | 风险 | 价值 |
|------|------|------|------|------|
| **1. 静默失效修复** | A1–A7 | 约 40 行 | 极低 | **最高**。改完暗色模式、全部弹窗层级、全部表单错误提示、入群认证按钮、Toast 语义色立刻恢复正常 |
| **2. 发布评课链路** | B1–B4 | 半天 | 低 | 直接影响核心转化 |
| **3. 状态反馈收敛** | 统一为 `SkeletonCard` + `EmptyState` + 带重试的 ErrorState 一套；uniappx 抽 `ListState.vue`；资源模块 8 处 `catch` 停止吞错误码；院系侧栏错误状态按核验修正案拆分 | 1–2 天 | 中 | 消除三套以上并行实现 |
| **4. 减法清理** | 删零引用组件（注意保留 `review.filter.all`，见核验记录）；手搓 OTP 输入换成已有的 `OtpCodeInput`；清理白装依赖；补 prettier 配置终结 107:12 的缩进分裂 | 半天 | 极低 | 纯收益 |
| **5. 信息架构重构** | C1–C5 | 2–3 天 | 高（涉及产品形态） | 需先决策 |
| **6. admin / uniappx** | 拆三个巨型页面；补 i18n；修 `admission-policy` 零数据可用性；uniappx 建 token 与安全区适配 | 3–5 天 | 中 | 可独立推进 |

**批次 1 与 2 建议立即执行**：改动小、全部经逐行验证、风险低，且修完用户能立刻感知。

**批次 5 需要产品决策**（是否删除浮动导航、课程入口如何组织、账号设置是否合并），不应由实现方单方面决定。

### 验证方式

工具链完备（pnpm 10.32 / Node 24.14，`eslint` / `prettier` / `vue-tsc` / `playwright` 均已安装）：

```bash
cd clients
pnpm type-check:all && pnpm lint:all
pnpm test:web
pnpm test:e2e:web && pnpm test:e2e:admin && pnpm test:e2e:uni
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

A 类修复建议补充回归测试：

- A1：E2E 断言弹窗打开时顶栏不可点击（`inert` 属性或 `toBeDisabled`）；
- A2：单测断言 `system` 模式下 `<html>` 带 `.dark` 类，且 `dark:` 工具类实际命中；
- A3/A4：CI grep 规则，拦截 `@theme` 中不存在的 token 类名，以及只有 `border-<色>` 而无 `border` 宽度的组合；
- A6：删除重复 CSS 后跑 `admission` 相关 E2E，确认按钮样式仍在；
- A7：单测断言 Toast 警告图标的 `color` 解析结果不等于 `--color-text-muted`。

注意 `clients/web/src/modules/review/__tests__/ratingDisplayPolicy.test.ts` 在模块顶层 `readFileSync` 读取被判定为死代码的组件文件，删除组件前需同步处理该测试（详见对应条目的核验记录）。

### 工程约束建议

本次发现的多数问题源于**没有机械约束**，建议一并补上：

1. `clients/` 下没有任何 prettier 配置，`web/package.json` 无 `format` 脚本，eslint 配置无一条格式化规则——`.editorconfig` 规定 `.ts`/`.vue` 用 4 空格，实际 **107 个 `.vue` 用 2 空格、12 个用 4 空格**。建议以多数派（2 空格）为准修订 `.editorconfig`，并接入 prettier + CI 校验。
2. 未定义 token 类名（A4）、无宽度边框（A3）、裸 `z-<数字>`（A1）都可以用 grep 规则在 CI 拦住。
3. `eslint.config.mjs:74` 显式关闭了 `vuejs-accessibility/no-static-element-interactions`。目前全站只有 1 处 `div` + `@click`，**尚未造成实际伤害**，属于潜在风险；建议打开该规则并对那 1 处加豁免注释，防止后续扩散。

---

## 8. 附：本报告未覆盖的范围

- 后端、CI/CD、部署与可观测性；
- 视觉设计的主观审美评价（配色是否好看、品牌调性是否恰当）——本报告只判断**一致性**与**可访问性**，不判断品味；
- 真实浏览器截图比对与跨浏览器兼容性测试（评审基于源码静态分析，未启动 dev server 做视觉回归）。**这意味着 A 类"静默失效"结论虽经逐行代码推演，仍建议在真实浏览器中抽验一次**；
- 性能实测（未做 Lighthouse / bundle 分析，仅从源码指出 `echarts` 在 `web/src` 零引用、`element-plus` 仅 3 处引用等结构性冗余）。
