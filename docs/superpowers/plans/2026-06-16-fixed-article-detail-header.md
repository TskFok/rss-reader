# Fixed Article Detail Header Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep the article detail title and actions fixed while only the article body scrolls.

**Architecture:** Split the existing detail dock into a fixed header and an inner scroll container. Move scroll reset behavior from the outer dock to the inner scroll container while preserving current article selection behavior.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, CSS.

---

### Task 1: Add Regression Test

**Files:**
- Modify: `web/src/pages/Home.test.tsx`

- [ ] **Step 1: Write the failing test**

Add a test that renders `Home`, opens the long article, finds `.article-detail-scroll`, and asserts `.article-detail-title` is outside that scroll container:

```tsx
test('文章详情标题不放在正文滚动容器内', async () => {
  const user = userEvent.setup();
  const store = new Map<string, string>();
  // @ts-expect-error test polyfill
  globalThis.localStorage = {
    getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
    setItem: (k: string, v: string) => {
      store.set(k, String(v));
    },
    removeItem: (k: string) => {
      store.delete(k);
    },
  };

  const { container } = render(
    <MemoryRouter initialEntries={['/']}>
      <ThemeProvider>
        <ToastProvider>
          <Home />
        </ToastProvider>
      </ThemeProvider>
    </MemoryRouter>
  );

  await user.click(await screen.findByRole('button', { name: /长文/ }));

  const scroll = container.querySelector('.article-detail-scroll') as HTMLDivElement;
  const title = container.querySelector('.article-detail-title') as HTMLAnchorElement;
  expect(scroll).not.toBeNull();
  expect(title).not.toBeNull();
  expect(scroll.contains(title)).toBe(false);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test -- Home.test.tsx -t 文章详情标题不放在正文滚动容器内`

Expected: FAIL because `.article-detail-scroll` does not exist yet.

### Task 2: Split Detail Header And Scroll Area

**Files:**
- Modify: `web/src/pages/Home.tsx`
- Modify: `web/src/index.css`

- [ ] **Step 1: Update markup and refs**

In `Home.tsx`, keep `.article-detail-header` as a direct child of `.article-detail-dock`, then add `.article-detail-scroll` around `.article-detail-meta` and `ArticleDetailContent`. The existing `detailDockScrollRef` should point to `.article-detail-scroll`.

- [ ] **Step 2: Update CSS**

In `index.css`, change `.article-detail-dock` to a column flex container with `overflow: hidden`. Add `.article-detail-scroll` with `overflow: auto`, `min-height: 0`, and `flex: 1`. Keep existing padding/background behavior consistent with the current panel.

- [ ] **Step 3: Run targeted test**

Run: `npm test -- Home.test.tsx -t 文章详情标题不放在正文滚动容器内`

Expected: PASS.

### Task 3: Verify Existing Scroll Reset Behavior

**Files:**
- Test: `web/src/pages/Home.test.tsx`

- [ ] **Step 1: Run Home tests**

Run: `npm test -- Home.test.tsx`

Expected: PASS, including existing tests for resetting article detail scroll position when switching feeds and articles.
