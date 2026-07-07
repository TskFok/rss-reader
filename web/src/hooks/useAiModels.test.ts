import { act, renderHook, waitFor } from '@testing-library/react';
import {
  clearAiModelsCache,
  ensureAiModelsLoaded,
  getAiModelsCacheForTest,
  refreshAiModelsCache,
  setAiModelsCache,
  useAiModels,
} from './useAiModels';
import { aiModelsApi } from '../api/client';

const { testModel } = vi.hoisted(() => ({
  testModel: {
    id: 1,
    name: 'test',
    base_url: 'https://api.example/v1',
    user_id: 1,
    created_at: '',
    updated_at: '',
  },
}));

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    aiModelsApi: {
      ...actual.aiModelsApi,
      list: vi.fn().mockResolvedValue({ data: [testModel] }),
    },
  };
});

beforeEach(() => {
  clearAiModelsCache();
  vi.mocked(aiModelsApi.list).mockClear();
});

test('ensureAiModelsLoaded 只请求一次', async () => {
  await ensureAiModelsLoaded();
  await ensureAiModelsLoaded();
  expect(aiModelsApi.list).toHaveBeenCalledTimes(1);
  expect(getAiModelsCacheForTest().models).toEqual([testModel]);
});

test('setAiModelsCache 写入后不再请求', async () => {
  setAiModelsCache([testModel]);
  await ensureAiModelsLoaded();
  expect(aiModelsApi.list).not.toHaveBeenCalled();
});

test('refreshAiModelsCache 会重新请求', async () => {
  setAiModelsCache([testModel]);
  await refreshAiModelsCache();
  expect(aiModelsApi.list).toHaveBeenCalledTimes(1);
});

test('clearAiModelsCache 清空后再次加载会请求', async () => {
  setAiModelsCache([testModel]);
  clearAiModelsCache();
  await ensureAiModelsLoaded();
  expect(aiModelsApi.list).toHaveBeenCalledTimes(1);
});

test('useAiModels 随缓存更新', async () => {
  const { result } = renderHook(() => useAiModels());
  expect(result.current.models).toEqual([]);

  await act(async () => {
    await ensureAiModelsLoaded();
  });

  await waitFor(() => {
    expect(result.current.models).toEqual([testModel]);
    expect(result.current.loaded).toBe(true);
  });
});
