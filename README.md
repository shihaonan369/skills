# Personal Skills

个人自研 skills 的唯一源码仓库。今后在这里开发，通过软链接接入各 agent，避免把源码散落在应用配置目录。

## 布局

每个 skill 使用 `<name>/SKILL.md` 作为入口，脚本、资源、文档和测试在同目录内自包含。

| Skill | 状态 | 说明 |
| --- | --- | --- |
| [agent-dashboard](agent-dashboard/README.md) | 本地可运行；Mac 已验证，其他平台仅交叉编译 | 可组合 TUI Panels + Claude 遥测；调用方决定布局，附独立 AI Hero 示例 |

## 本机链接

```sh
ln -s "$HOME/source/skills/agent-dashboard" "$HOME/.claude/skills/agent-dashboard"
```

仅创建各自的 skill 链接，保留 `~/.claude/skills/` 中现有的其他 skills。运行时状态与凭证不入库。

通用 dashboard 的历史交接见 [HANDOFF.md](HANDOFF.md)；当前接口见 [Panel 文档](agent-dashboard/docs/PANELS.md)，验证范围见 [验证矩阵](agent-dashboard/docs/VALIDATION.md)。
