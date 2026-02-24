export interface CatalogModel {
  modelId: string
  name: string
  contextWindow: number
  maxTokens: number
  reasoning: boolean
  isMultimodal: boolean
}

export interface ProviderInfo {
  label: string
  defaultBaseUrl: string
  models: CatalogModel[]
}

export const providerCatalog: Record<string, ProviderInfo> = {
  openai: {
    label: 'OpenAI',
    defaultBaseUrl: 'https://api.openai.com/v1',
    models: [
      { modelId: 'gpt-4o', name: 'GPT-4o', contextWindow: 128000, maxTokens: 16384, reasoning: false, isMultimodal: true },
      { modelId: 'gpt-4o-mini', name: 'GPT-4o Mini', contextWindow: 128000, maxTokens: 16384, reasoning: false, isMultimodal: true },
      { modelId: 'gpt-4-turbo', name: 'GPT-4 Turbo', contextWindow: 128000, maxTokens: 4096, reasoning: false, isMultimodal: true },
      { modelId: 'o1', name: 'o1', contextWindow: 200000, maxTokens: 100000, reasoning: true, isMultimodal: true },
      { modelId: 'o1-pro', name: 'o1 Pro', contextWindow: 200000, maxTokens: 100000, reasoning: true, isMultimodal: true },
      { modelId: 'o3', name: 'o3', contextWindow: 200000, maxTokens: 100000, reasoning: true, isMultimodal: true },
      { modelId: 'o3-mini', name: 'o3 Mini', contextWindow: 200000, maxTokens: 100000, reasoning: true, isMultimodal: false },
      { modelId: 'o3-pro', name: 'o3 Pro', contextWindow: 200000, maxTokens: 100000, reasoning: true, isMultimodal: true },
    ],
  },
  'openai-compat': {
    label: 'OpenAI Compatible',
    defaultBaseUrl: '',
    models: [],
  },
  anthropic: {
    label: 'Anthropic',
    defaultBaseUrl: 'https://api.anthropic.com/v1',
    models: [
      { modelId: 'claude-opus-4-6', name: 'Claude Opus 4.6', contextWindow: 200000, maxTokens: 128000, reasoning: true, isMultimodal: true },
      { modelId: 'claude-sonnet-4-5-20250929', name: 'Claude Sonnet 4.5', contextWindow: 200000, maxTokens: 64000, reasoning: true, isMultimodal: true },
      { modelId: 'claude-haiku-4-5-20251001', name: 'Claude Haiku 4.5', contextWindow: 200000, maxTokens: 64000, reasoning: true, isMultimodal: true },
      { modelId: 'claude-3-7-sonnet-20250219', name: 'Claude 3.7 Sonnet', contextWindow: 200000, maxTokens: 64000, reasoning: true, isMultimodal: true },
      { modelId: 'claude-3-5-sonnet-20241022', name: 'Claude 3.5 Sonnet', contextWindow: 200000, maxTokens: 8192, reasoning: false, isMultimodal: true },
      { modelId: 'claude-3-5-haiku-20241022', name: 'Claude 3.5 Haiku', contextWindow: 200000, maxTokens: 8192, reasoning: false, isMultimodal: true },
    ],
  },
  google: {
    label: 'Google',
    defaultBaseUrl: 'https://generativelanguage.googleapis.com/v1beta',
    models: [
      { modelId: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro', contextWindow: 1048576, maxTokens: 65536, reasoning: true, isMultimodal: true },
      { modelId: 'gemini-2.5-flash', name: 'Gemini 2.5 Flash', contextWindow: 1048576, maxTokens: 65536, reasoning: true, isMultimodal: true },
      { modelId: 'gemini-2.0-flash', name: 'Gemini 2.0 Flash', contextWindow: 1048576, maxTokens: 8192, reasoning: false, isMultimodal: true },
      { modelId: 'gemini-1.5-pro', name: 'Gemini 1.5 Pro', contextWindow: 1000000, maxTokens: 8192, reasoning: false, isMultimodal: true },
      { modelId: 'gemini-1.5-flash', name: 'Gemini 1.5 Flash', contextWindow: 1000000, maxTokens: 8192, reasoning: false, isMultimodal: true },
    ],
  },
  azure: {
    label: 'Azure OpenAI',
    defaultBaseUrl: '',
    models: [],
  },
  bedrock: {
    label: 'Amazon Bedrock',
    defaultBaseUrl: 'us-east-1',
    models: [],
  },
  mistral: {
    label: 'Mistral',
    defaultBaseUrl: 'https://api.mistral.ai/v1',
    models: [
      { modelId: 'mistral-large-latest', name: 'Mistral Large', contextWindow: 262144, maxTokens: 262144, reasoning: false, isMultimodal: true },
      { modelId: 'mistral-medium-latest', name: 'Mistral Medium', contextWindow: 128000, maxTokens: 16384, reasoning: false, isMultimodal: true },
      { modelId: 'mistral-small-latest', name: 'Mistral Small', contextWindow: 128000, maxTokens: 16384, reasoning: false, isMultimodal: true },
      { modelId: 'codestral-latest', name: 'Codestral', contextWindow: 256000, maxTokens: 4096, reasoning: false, isMultimodal: false },
    ],
  },
  xai: {
    label: 'xAI',
    defaultBaseUrl: 'https://api.x.ai/v1',
    models: [
      { modelId: 'grok-4', name: 'Grok 4', contextWindow: 256000, maxTokens: 256000, reasoning: true, isMultimodal: true },
      { modelId: 'grok-3', name: 'Grok 3', contextWindow: 131072, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'grok-3-mini', name: 'Grok 3 Mini', contextWindow: 131072, maxTokens: 8192, reasoning: true, isMultimodal: false },
    ],
  },
  ollama: {
    label: 'Ollama',
    defaultBaseUrl: 'http://localhost:11434/v1',
    models: [],
  },
  dashscope: {
    label: 'DashScope / 通义千问',
    defaultBaseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    models: [
      { modelId: 'qwen-max', name: 'Qwen Max', contextWindow: 32768, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'qwen-plus', name: 'Qwen Plus', contextWindow: 131072, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'qwen-turbo', name: 'Qwen Turbo', contextWindow: 131072, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'qwen-vl-max', name: 'Qwen VL Max', contextWindow: 32768, maxTokens: 4096, reasoning: false, isMultimodal: true },
      { modelId: 'qwq-32b', name: 'QwQ 32B', contextWindow: 131072, maxTokens: 8192, reasoning: true, isMultimodal: false },
    ],
  },
  deepseek: {
    label: 'DeepSeek',
    defaultBaseUrl: 'https://api.deepseek.com',
    models: [
      { modelId: 'deepseek-chat', name: 'DeepSeek V3', contextWindow: 163840, maxTokens: 65536, reasoning: false, isMultimodal: false },
      { modelId: 'deepseek-reasoner', name: 'DeepSeek R1', contextWindow: 163840, maxTokens: 65536, reasoning: true, isMultimodal: false },
    ],
  },
  'zai-global': {
    label: 'Z.AI / 智谱 (Global)',
    defaultBaseUrl: 'https://api.z.ai/api/paas/v4',
    models: [
      { modelId: 'glm-5', name: 'GLM-5', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7', name: 'GLM-4.7', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flash', name: 'GLM-4.7 Flash', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flashx', name: 'GLM-4.7 FlashX', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
    ],
  },
  'zai-cn': {
    label: 'Z.AI / 智谱 (国内)',
    defaultBaseUrl: 'https://open.bigmodel.cn/api/paas/v4',
    models: [
      { modelId: 'glm-5', name: 'GLM-5', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7', name: 'GLM-4.7', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flash', name: 'GLM-4.7 Flash', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flashx', name: 'GLM-4.7 FlashX', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
    ],
  },
  'zai-coding-global': {
    label: 'Z.AI / 智谱 Coding (Global)',
    defaultBaseUrl: 'https://api.z.ai/api/coding/paas/v4',
    models: [
      { modelId: 'glm-4.7', name: 'GLM-4.7', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flash', name: 'GLM-4.7 Flash', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flashx', name: 'GLM-4.7 FlashX', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
    ],
  },
  'zai-coding-cn': {
    label: 'Z.AI / 智谱 Coding (国内)',
    defaultBaseUrl: 'https://open.bigmodel.cn/api/coding/paas/v4',
    models: [
      { modelId: 'glm-4.7', name: 'GLM-4.7', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flash', name: 'GLM-4.7 Flash', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
      { modelId: 'glm-4.7-flashx', name: 'GLM-4.7 FlashX', contextWindow: 204800, maxTokens: 131072, reasoning: true, isMultimodal: false },
    ],
  },
  'minimax-global': {
    label: 'MiniMax (Global)',
    defaultBaseUrl: 'https://api.minimax.io/v1',
    models: [
      { modelId: 'MiniMax-M2.5', name: 'MiniMax M2.5', contextWindow: 200000, maxTokens: 8192, reasoning: true, isMultimodal: false },
      { modelId: 'MiniMax-M2.5-Lightning', name: 'MiniMax M2.5 Lightning', contextWindow: 200000, maxTokens: 8192, reasoning: true, isMultimodal: false },
      { modelId: 'MiniMax-M2.1', name: 'MiniMax M2.1', contextWindow: 200000, maxTokens: 8192, reasoning: false, isMultimodal: false },
    ],
  },
  'minimax-cn': {
    label: 'MiniMax (国内)',
    defaultBaseUrl: 'https://api.minimaxi.com/v1',
    models: [
      { modelId: 'MiniMax-M2.5', name: 'MiniMax M2.5', contextWindow: 200000, maxTokens: 8192, reasoning: true, isMultimodal: false },
      { modelId: 'MiniMax-M2.5-Lightning', name: 'MiniMax M2.5 Lightning', contextWindow: 200000, maxTokens: 8192, reasoning: true, isMultimodal: false },
      { modelId: 'MiniMax-M2.1', name: 'MiniMax M2.1', contextWindow: 200000, maxTokens: 8192, reasoning: false, isMultimodal: false },
    ],
  },
  'moonshot-global': {
    label: 'Moonshot / Kimi (Global)',
    defaultBaseUrl: 'https://api.moonshot.ai/v1',
    models: [
      { modelId: 'kimi-k2.5', name: 'Kimi K2.5', contextWindow: 256000, maxTokens: 8192, reasoning: false, isMultimodal: false },
    ],
  },
  'moonshot-cn': {
    label: 'Moonshot / Kimi (国内)',
    defaultBaseUrl: 'https://api.moonshot.cn/v1',
    models: [
      { modelId: 'kimi-k2.5', name: 'Kimi K2.5', contextWindow: 256000, maxTokens: 8192, reasoning: false, isMultimodal: false },
    ],
  },
  volcengine: {
    label: '火山引擎 / Doubao',
    defaultBaseUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    models: [
      { modelId: 'doubao-1.5-pro-256k', name: 'Doubao 1.5 Pro 256K', contextWindow: 256000, maxTokens: 16384, reasoning: false, isMultimodal: false },
      { modelId: 'doubao-1.5-pro-32k', name: 'Doubao 1.5 Pro 32K', contextWindow: 32768, maxTokens: 4096, reasoning: false, isMultimodal: false },
      { modelId: 'doubao-1.5-lite-128k', name: 'Doubao 1.5 Lite 128K', contextWindow: 128000, maxTokens: 16384, reasoning: false, isMultimodal: false },
      { modelId: 'doubao-1.5-vision-pro', name: 'Doubao 1.5 Vision Pro', contextWindow: 32768, maxTokens: 4096, reasoning: false, isMultimodal: true },
    ],
  },
  'volcengine-coding': {
    label: '火山引擎 Coding',
    defaultBaseUrl: 'https://ark.cn-beijing.volces.com/api/coding/v3',
    models: [
      { modelId: 'doubao-seed-code-preview-251028', name: 'Doubao Seed Code', contextWindow: 256000, maxTokens: 16384, reasoning: false, isMultimodal: false },
    ],
  },
  qianfan: {
    label: '百度千帆',
    defaultBaseUrl: 'https://qianfan.baidubce.com/v2',
    models: [
      { modelId: 'deepseek-v3.2', name: 'DeepSeek V3.2', contextWindow: 98304, maxTokens: 32768, reasoning: true, isMultimodal: false },
      { modelId: 'ernie-4.5-turbo-vl-32k', name: 'ERNIE 4.5 Turbo VL', contextWindow: 32768, maxTokens: 8192, reasoning: false, isMultimodal: true },
    ],
  },
  groq: {
    label: 'Groq',
    defaultBaseUrl: 'https://api.groq.com/openai/v1',
    models: [
      { modelId: 'llama-3.3-70b-versatile', name: 'Llama 3.3 70B', contextWindow: 131072, maxTokens: 32768, reasoning: false, isMultimodal: false },
      { modelId: 'llama-3.1-8b-instant', name: 'Llama 3.1 8B', contextWindow: 131072, maxTokens: 131072, reasoning: false, isMultimodal: false },
      { modelId: 'deepseek-r1-distill-llama-70b', name: 'DeepSeek R1 Distill 70B', contextWindow: 131072, maxTokens: 8192, reasoning: true, isMultimodal: false },
    ],
  },
  openrouter: {
    label: 'OpenRouter',
    defaultBaseUrl: 'https://openrouter.ai/api/v1',
    models: [],
  },
  together: {
    label: 'Together AI',
    defaultBaseUrl: 'https://api.together.xyz/v1',
    models: [],
  },
  fireworks: {
    label: 'Fireworks AI',
    defaultBaseUrl: 'https://api.fireworks.ai/inference/v1',
    models: [],
  },
  perplexity: {
    label: 'Perplexity',
    defaultBaseUrl: 'https://api.perplexity.ai',
    models: [
      { modelId: 'sonar-pro', name: 'Sonar Pro', contextWindow: 200000, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'sonar', name: 'Sonar', contextWindow: 128000, maxTokens: 8192, reasoning: false, isMultimodal: false },
      { modelId: 'sonar-reasoning-pro', name: 'Sonar Reasoning Pro', contextWindow: 128000, maxTokens: 8192, reasoning: true, isMultimodal: false },
    ],
  },
}

export function getProviderModels(clientType: string): CatalogModel[] {
  return providerCatalog[clientType]?.models ?? []
}

export function getProviderDefaultBaseUrl(clientType: string): string {
  return providerCatalog[clientType]?.defaultBaseUrl ?? ''
}

export function getProviderLabel(clientType: string): string {
  return providerCatalog[clientType]?.label ?? clientType
}

export function getAllClientTypes(): string[] {
  return Object.keys(providerCatalog)
}
