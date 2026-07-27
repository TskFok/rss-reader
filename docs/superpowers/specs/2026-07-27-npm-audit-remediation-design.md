# npm audit 漏洞修复设计

## 目标

消除可由已发布依赖版本修复的 `npm audit` 高危漏洞，同时保持现有页面路由行为可用。

## 根因与范围

- ESLint 9 的传递依赖包含受影响的 `minimatch` 与 `brace-expansion`；升级 `eslint` 至 10.8.0 后可消除这组告警。
- `react-router-dom` 的安全公告版本范围相互冲突：一条公告将 7.12.0 至 8.2.0 标为受影响，另一组公告将 7.11.0 及更低版本标为受影响。npm 仓库当前最高仅发布至 7.18.1，审计建议的 8.3.0 不存在。
- 仅修改 `web/package.json`、`web/package-lock.json` 和因 ESLint 10 兼容性实际需要的配置文件；不更改产品功能或路由结构。

## 实施方案

1. 将 ESLint 及其直接相关的开发依赖升级到兼容 ESLint 10 的版本，修复其传递依赖漏洞。
2. 将 `react-router-dom` 更新为当前可发布的最高版本 7.18.1。
3. 使用 npm 更新锁文件，保留现有锁文件中与本任务有关的安全补丁。
4. 运行 lint、单元测试、构建与 `npm audit`；若升级引发配置或 API 不兼容，仅进行必要的最小适配。

## 验收条件

- `npm audit --json` 中不再包含 ESLint 传递依赖漏洞；React Router 的剩余告警须等待上游发布可用修复版本。
- `npm test` 和 `npm run build` 均成功退出。
- `npm run lint` 的新增 hooks 插件规则会报告现有代码问题；这些问题不在本次依赖安全更新的修改范围内。
- 现有 React Router v7 的声明式路由、`MemoryRouter` 测试工具与导航钩子保持可用。
