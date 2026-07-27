# npm audit 漏洞修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Web 前端依赖解析至当前 npm 仓库可提供的最安全版本，并验证既有构建和路由测试。

**Architecture:** 通过 package manifest 声明 ESLint 10 及兼容插件版本，并将 React Router DOM 更新到当前可发布的 7.18.1。npm 负责生成完整、可复现的 package-lock；不改变应用路由实现，兼容性由现有测试与构建检验。

**Tech Stack:** npm、Vite、React 19、React Router、ESLint、Vitest、TypeScript。

## Global Constraints

- ESLint 目标版本为 10.8.0。
- `react-router-dom` 必须为当前已发布的 7.18.1。
- React Router 审计建议的 8.3.0 当前未发布；不得使用不存在的版本或以 `--force` 隐藏审计告警。
- 不修改应用功能或路由结构，除非升级验证显示存在直接兼容性错误。

---

## 文件结构

- 修改：`web/package.json` — 固定直接依赖与开发依赖版本。
- 修改：`web/package-lock.json` — 锁定已解析的安全依赖树。
- 按需修改：`web/eslint.config.js` — 仅在 ESLint 10 报告配置不兼容时调整。

### Task 1: 更新受影响的直接依赖

**Files:**
- Modify: `web/package.json`
- Modify: `web/package-lock.json`

**Interfaces:**
- Consumes: npm registry 中 ESLint 10.8.0、兼容 ESLint 10 的插件版本，以及 `react-router-dom@7.18.1`。
- Produces: 可由 `npm ci` 重建的最新可用依赖树。

- [ ] **Step 1: 记录当前漏洞基线**

Run: `npm audit --json`

Expected: 7 个高危漏洞，分别来自 ESLint 传递依赖和 React Router。

- [ ] **Step 2: 更新 manifest 与锁文件**

Run: `npm install -D eslint@10.8.0 @eslint/js@10.0.1 typescript-eslint@8.65.0 eslint-plugin-react-hooks@7.1.1 eslint-plugin-react-refresh@0.5.3 && npm install react-router-dom@7.18.1`

Expected: `package.json` 记录目标直接依赖，`package-lock.json` 中的依赖树同步更新。

- [ ] **Step 3: 验证安全修复**

Run: `npm audit --json`

Expected: ESLint 传递依赖漏洞消失；若 React Router 仍有告警，记录 npm 安全数据库与已发布版本之间的冲突。

### Task 2: 验证开发工具链与应用兼容性

**Files:**
- Modify if required: `web/eslint.config.js`
- Test: `web/src/**/*.test.tsx`

**Interfaces:**
- Consumes: Task 1 生成的安全依赖树。
- Produces: 可通过 lint、单元测试和生产构建的前端工程。

- [ ] **Step 1: 检查 ESLint 配置兼容性**

Run: `npm run lint`

Expected: ESLint 10 正常加载 `eslint.config.js`；若新增规则报告历史代码问题，记录其规则名称和数量，不在本任务中重构业务代码。

- [ ] **Step 2: 最小化修复配置（仅在 Step 1 失败时）**

调整 `eslint.config.js` 中被 ESLint 10 明确报错的配置项，不调整规则语义或应用源代码。

- [ ] **Step 3: 回归验证**

Run: `npm test && npm run build && npm audit --json`

Expected: 单元测试和构建成功；审计报告仅可能保留 React Router 上游无可用修复版本导致的告警。
