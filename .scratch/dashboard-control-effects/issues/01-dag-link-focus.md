# DAG 链路聚焦应该突出哪些关系

Type: prototype
Labels: wayfinder:prototype
Mode: HITL
Status: resolved
Assignee: Pi (local prototype session)
Map: [Agent Dashboard 控件与动效路线图](../map.md)
Blocked by: none

## Question

用带明确 demo 标识的低保真草图或一次性原型，与用户决定 DAG 链路聚焦的交互：如何选择节点，直接邻居与上下游路径如何切换，以及如何区分入边、出边、回边与无关节点？
必须讨论双向边、环、孤立节点及未绑定观测状态；只根据声明的图表达关系，不把高亮解释成实际通信。确认不只依赖颜色的识别方式。资产另存并从本票链接，须经用户反馈才能解决。

## Assets

- [三方案交互原型](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/dag-link-focus.prototype.html)
- [运行方式、试用场景与验证边界](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/README.md)
- 截图：[原图聚焦](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-A.png)、[关系侧栏](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-B.png)、[关系账本](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-C.png)。

## Comments

Pi 已认领并制作只读 synthetic 原型。三个方案均通过 JavaScript 语法与 Chrome headless 初次渲染检查；不代表终端运行验证或完整交互验收。

提议 A 作为默认基础，比较是否需要 B 的侧栏或 C 的关系账本。关系计算可在“直接邻居 / 所有上游 / 所有下游 / 上下游合看”之间切换，return 可选择是否参与聚焦，声明边本身不删除。双向边、自回边、环、孤立节点及 unknown/declared-only 状态均提供示例。

原型交付时等待用户反馈，未形成 Answer。随后用户回复“按推荐”，确认下述视觉选择；不是完整交互验收，也不是产品实现授权。

## Answer

用户对“A 做默认，按需展开 B 的侧栏”的推荐回复“按推荐”。

- 采用 A 的原图就地聚焦作为默认视觉方向，保留图的空间结构，无关项降低强调。
- B 的关系侧栏按需展开，不默认常驻；C 不作为默认布局。
- 选择理由沿用本轮推荐：默认保留整图空间感，需要时再查看节点和关系细节。
- 原型中的未知状态、声明关系与演示标签不因选中而变为真实活动证明。

本票以**视觉方向选择**收束。原始问题中的终端节点选择、范围默认值、return 参与规则及窄窗口下侧栏行为仍未获逐项批准，已明确拆至 [DAG 链路聚焦在终端里如何选择节点和展开侧栏](07-terminal-link-focus-contract.md)；不能据此宣称交互契约已完整确定。没有改动 Go 渲染器、产品 schema、采集或安装配置，也没有授权直接推广原型代码。
