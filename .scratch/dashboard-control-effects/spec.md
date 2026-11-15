# 已授权执行：DAG 链路聚焦与状态回响

## Scope

用户在确认推荐规则后要求“没，开工”。实施原生 Go TUI 的两项已选方向；不引入新运行时、不改变遥测/schema、不动应用配置或真实会话，不实现面板放大、时间线、事件瀑布。

## Reversible implementation defaults (not yet human-accepted terminal UX)

- DAG 仍用箭头平移/滚动；[ / ] 按声明顺序选节点，不自动平移；c 清除选择。
- f 在直接邻居/所有上游/所有下游/合看间循环；默认直接邻居；r 切换 return 参与（默认参与，边本身不删除）。
- i 切换叶面板内部关系侧栏，足够宽时并排，窄时提供明确提示而不重排调用方布局。选择和侧栏均为查看器本地状态。
- x 清除本地变化提示。行前保留两个字符用于短暂/静态标记，动效不会改变布局尺寸。
- 行差异以真实快照字段比较；1.2 秒短暂强调，摘要保留到下一批真实字段变化、重建基线或主动清除。
- 暂停丢弃在途快照；恢复立即请求新基线且不补闪，过期的在途快照不能覆盖恢复后的显示。
- 历史/帮助视图抑制强调；事件历史保持阅读锚点。新布局移除选中节点时清除本地节点选择。

## Implementation and validation

Implemented in native `link_focus.go`, `echo.go`, and the existing DAG/layout/viewer integration. No new dependencies or persisted/schema fields.

- Full Go race/coverage suite: 51 top-level + 15 subtests passed; vet passed. Core coverage 91.3%.
- Nine Mac PTY runs, including keyboard focus/sidebar, real synthetic task row changes, motion-off markers, resume baseline, SIGWINCH and both exit keys.
- Extended parser/viewer fuzz: 5 seconds, 2 workers, 46,387 executions, no failure.
- Six platform archives rebuilt; all 12 binary/archive SHA-256 entries and single-binary package contents verified; extracted native executable starts. Linux/Windows/Intel-Mac are cross-build-only.
- Separate Ghostty-owned TTY opened with the labelled synthetic preview launcher. Temporary state is isolated and cleans on exit; no actual Claude run or settings changes.
- Exact evidence/boundaries are in [VALIDATION.md](../../agent-dashboard/docs/VALIDATION.md). No commit/push/publish, remote host or real-user telemetry mutation.

Terminal-interaction defaults are implemented under the explicit execution authorization but remain available for feedback in the separate acceptance ticket. This does not complete the broader roadmap or authorize remaining controls.
