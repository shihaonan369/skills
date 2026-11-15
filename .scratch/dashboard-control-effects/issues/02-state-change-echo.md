# 状态变更回响应该强调哪些真实变化

Type: prototype
Labels: wayfinder:prototype
Mode: HITL
Status: resolved
Assignee: Pi (local state-echo prototype session)
Map: [Agent Dashboard 控件与动效路线图](../map.md)
Blocked by: none

## Question

用带 demo 标识的状态前后对照或一次性原型，与用户决定哪些真实变化触发一次性强调，强调在哪个局部出现、持续多久，以及多个变化同时出现如何保持可读？
明确初次加载、重新打开、重复快照、滚动历史、暂停和恢复时是否触发；区分单个工具错误与整个 Agent 失败。无动画模式应保留文字或标记，不抢焦点、滚动或改变尺寸。资产另存并链接，须经用户反馈才能解决。

## Assets

- [可点按的状态规则原型](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/state-change-echo.prototype.html)
- [初始状态截图](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/state-change-echo.png)
- [运行方式与验证边界](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/README.md)

## Comments

用户在上一项视觉选择记录后回复“嗯”，本会话接续认领本票，只推进这一项。
原型为单文件 HTML，以纯 reducer 和内存合成源表达规则，不连接真实遥测或产品。提供自由试用及五组引导：工具错误不等于 Agent 失败、暂停后不补闪、重复与重新打开、静态与历史、同批多变化。

当前提案（**尚待用户确认**）：
- 只强调发生变化的行，不让单个工具错误把整个 Agent 卡片变成失败；并行工具独立保留。
- 短暂强调模拟 1200 ms，时钟只能手动推进；文字变化摘要保留至下一批变化、基线重建或主动清除。
- 初次加载、重新打开及重复同一快照不产生新强调，重复快照不延长强调期限。
- 暂停冻结显示，合成源可继续变化；恢复采用最新基线，不补闪，也不积累回响队列。
- 关闭效果仍展示状态和静态摘要；模拟查看历史抑制新强调，不重播旧变化。
- 只有 task 的明确 completed 更新该任务，不把 returned / stopped 当作完成。

已做 JavaScript 语法检查、macOS Chrome headless 初次渲染并查看截图；随后限定卡片和字段高度，避免内容变化导致草图扩张。未进行正式终端集成、真实动画计时、跨平台运行或完整浏览器交互验收。截图为初始状态，不能作为变化动效证据。

原型交付时等待反馈；随后用户回复“没，开工”，接受推荐规则并授权正式实现。

## Answer

采用本票 Comments 中的推荐规则：仅强调改变的行；工具错误与 Agent 生命周期分离；重复快照不延长强调；初次加载/重开/恢复建立基线、不补闪；关闭效果保留静态摘要。短暂强调以 1.2 秒作为首版实现默认值，不是已完成的计时验收。

用户明确授权实施已确认的 DAG 链路聚焦与状态回响，未授权把面板放大或新数据控件一并实现。执行与验证记录另见本 effort 的 spec.md；本票只保存规则决策。
