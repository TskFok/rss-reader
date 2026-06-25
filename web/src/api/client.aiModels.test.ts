import { describe, expect, it, vi } from 'vitest';

const putMock = vi.fn().mockResolvedValue({ data: {} });

vi.mock('axios', () => ({
  default: {
    create: () => ({
      put: putMock,
      get: vi.fn(),
      post: vi.fn(),
      delete: vi.fn(),
      interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
    }),
  },
}));

describe('aiModelsApi.update backup_model_id', () => {
  it('清除备用模型时发送 backup_model_id: 0', async () => {
    putMock.mockClear();
    const { aiModelsApi } = await import('./client');
    await aiModelsApi.update(1, 'name', 'https://api.example/v1', undefined, null);
    expect(putMock).toHaveBeenCalledWith('/ai-models/1', {
      name: 'name',
      base_url: 'https://api.example/v1',
      backup_model_id: 0,
    });
  });

  it('未修改备用模型时不发送 backup_model_id', async () => {
    putMock.mockClear();
    const { aiModelsApi } = await import('./client');
    await aiModelsApi.update(1, 'name', 'https://api.example/v1');
    expect(putMock).toHaveBeenCalledWith('/ai-models/1', {
      name: 'name',
      base_url: 'https://api.example/v1',
    });
  });
});
