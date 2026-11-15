# 这份控件路线图的顺序与验收门槛是什么

Type: grilling
Labels: wayfinder:grilling
Mode: HITL
Status: open
Assignee: none
Map: [Agent Dashboard 控件与动效路线图](../map.md)
Blocked by: 01, 02, 03, 05, 07

依赖：[DAG 链路聚焦应该突出哪些关系](01-dag-link-focus.md)、[状态变更回响应该强调哪些真实变化](02-state-change-echo.md)、[面板一键放大的进入与返回契约是什么](03-panel-zoom.md)、[泳道时间线与事件折叠瀑布是否进入下一批](05-next-controls-choice.md)、[DAG 链路聚焦在终端里如何选择节点和展开侧栏](07-terminal-link-focus-contract.md)。

## Question

在各增强的方案和下一批取舍明确后，与用户确定可交接的优先顺序和验收门槛：哪些先做、哪些可独立做，如何验证静态/动画/暂停一致性、小窗口与缩放、键盘可达性、终端字符宽度以及性能开销？
先读取当前基线证据再提可测门槛，不把跨平台编译当作运行验证，不承诺未经测量的帧率。明确最终路线图的交接位置；如果仍有实质决策缺口，先创建并关联后续票，不提前宣布路线图完成。
