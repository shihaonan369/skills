# Throwaway: 控件决策原型

这是决策原型，不是产品入口，不读取遥测，不控制 Agent，也不属于发行包。
浏览器只是承载字符草图，产品仍是原生终端 TUI。

## 打开

直接双击 `dag-link-focus.prototype.html`，或在仓库根目录：

```sh
open agent-dashboard/prototypes/dag-link-focus.prototype.html
```

不需要启动服务或安装包。单文件内的三种方案通过底部按钮切换，也支持 URL：
`?variant=A`、`?variant=B`、`?variant=C`。
浏览状态保存在内存；URL 仅记录方案、选中节点、范围与 return 开关，不保存遥测。

- **A 原图聚焦**：节点位置不动，无关项变暗；下方精确边清单。
- **B 原图 + 关系侧栏**：保留上下文亮度，单独说明选中节点与上下游。
- **C 关系账本优先**：关系清单优先，下方保留原图作参考。

## 建议试的场景

1. 选 Reviewer：unknown 不会因选中变成 running，自回边同时属于入/出关系。
2. 选 Lead 或 Relay：lead → relay 与 relay → lead 是两条独立声明，不合并成无向边。
3. 选 Relay，切到“上下游合看”：包含 return 时形成环，用有限可达集合表达，不枚举路径、不声称关键路径。
4. 取消包含 return：只改变聚焦计算，声明边仍在图里。
5. 选 Isolated note：仅选中自身，边清单为空，不推断未观测的关系。
6. 选 Gate annotation：无绑定保持 declared only，不伪造 Agent。
7. 清除聚焦：回到中性全图。

前缀与表格说明关系，不只依赖颜色。图中交叉不能作为节点；精确方向以边清单为准。
节点选择按钮支持键盘；图内点击只是浏览器便捷操作。正式终端的工程默认键位见 [DAG 合约](../docs/DAG.md#link-focus)，不能将浏览器左右切方案键当作产品键位。

## 验证与边界

- 已做 Node JavaScript 语法检查、macOS 本地 Chrome headless 三种方案的初次渲染检查。
- 已查看 A 的截图，修正字符粗体导致的列错位，以及回边穿过孤立注释节点的问题。
- 尚未做真实终端渲染、产品集成、辅助技术、Linux/Windows 运行验证；浏览器按钮交互尚待用户试用。
- 草图人工定坐标，不等于正式 DAG 的 30-cell 节点或自动布线结果。
- 用户已选 A 默认 + B 侧栏按需，并在后续“没，开工”时授权两项原生控件实现；规则与实际验证以产品 docs 为准。草图仍是一次性资产，在本原型分支归档；未直接推广为产品代码，也不包含在运行时发行包。

决策票：[DAG 链路聚焦应该突出哪些关系](../../.scratch/dashboard-control-effects/issues/01-dag-link-focus.md)。

## 状态变更回响：规则草图

双击 [state-change-echo.prototype.html](state-change-echo.prototype.html)，或运行：

```sh
open agent-dashboard/prototypes/state-change-echo.prototype.html
```

用上方自由试用按钮或下方五组逐步场景检查回响规则。所有源数据与显示状态都在内存中；只由按钮模拟工具、任务、Agent 状态和查看器操作。不向产品发布数据。

本原型重点是“哪些变化值得提示”，不是 UI 变体或动画质感；1200 ms 是提议值，必须点“时间 +400 ms”推进。默认不自动播放动画，实际持续时间、颜色和柔和过渡尚未验收。完整相关状态可展开查看。

卡片行只显示有限宽度，详细字段值保留在状态区；浏览器草图不等于正式终端的 cell 布局。
已做 Node 语法检查和 Chrome headless 初次渲染/截图检查；未宣称按钮完整交互、正式集成或跨平台运行验证。

用户已接受该原型推荐规则并授权正式实现；此文件仍只验证状态模型，不是动画或终端验收证据。原生实现的实际检查见 [VALIDATION.md](../docs/VALIDATION.md)。

决策票：[状态变更回响应该强调哪些真实变化](../../.scratch/dashboard-control-effects/issues/02-state-change-echo.md)。

