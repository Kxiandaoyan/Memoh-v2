export interface CatalogModel {
  modelId: string
  name: string
  contextWindow: number
  isMultimodal: boolean
  reasoning: boolean
  maxTokens: number
}

interface ProviderEntry {
  label: string
  defaultBaseUrl: string
  models: CatalogModel[]
}

export const providerCatalog: Record<string, ProviderEntry> = {
  openai: {
    label: 'OpenAI',
    defaultBaseUrl: 'https://api.openai.com/v1',
    models: [
      { modelId: 'gpt-4o', name: 'GPT-4o', contextWindow: 128000, isMultimodal: true, reasoning: false, maxTokens: 16384 },
      { modelId: 'gpt-4o-mini', name: 'GPT-4o Mini', contextWindow: 128000, isMultimodal: true, reasoning: false, maxTokens: 16384 },
      { modelId: 'o1', name: 'o1', contextWindow: 200000, isMultimodal: true, reasoning: true, maxTokens: 100000 },
      { modelId: 'o1-mini', name: 'o1 Mini', contextWindow: 128000, isMultimodal: false, reasoning: true, maxTokens: 65536 },
      { modelId: 'o3-mini', name: 'o3 Mini', contextWindow: 200000, isMultimodal: false, reasoning: true, maxTokens: 100000 },
    ],
  },
  'openai-compat': {
    label: 'OpenAI Compatible',
    defaultBaseUrl: '',
    models: [],
  },
  anthropic: {
    label: 'Anthropic',
    defaultBaseUrl: 'https://api.anthropic.com',
    models: [
      { modelId: 'claude-sonnet-4-20250514', name: 'Claude Sonnet 4', contextWindow: 200000, isMultimodal: true, reasoning: true, maxTokens: 16000 },
      { modelId: 'claude-3-5-haiku-20241022', name: 'Claude 3.5 Haiku', contextWindow: 200000, isMultimodal: false, reasoning: false, maxTokens: 8192 },
    ],
  },
  google: {
    label: 'Google',
    defaultBaseUrl: 'https://generativelanguage.googleapis.com/v1beta',
    models: [
      { modelId: 'gemini-2.5-flash', name: 'Gemini 2.5 Flash', contextWindow: 1048576, isMultimodal: true, reasoning: true, maxTokens: 65536 },
      { modelId: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro', contextWindow: 1048576, isMultimodal: true, reasoning: true, maxTokens: 65536 },
      { modelId: 'gemini-2.0-flash', name: 'Gemini 2.0 Flash', contextWindow: 1048576, isMultimodal: true, reasoning: false, maxTokens: 8192 },
    ],
  },
  azure: {
    label: 'Azure OpenAI',
    defaultBaseUrl: '',
    models: [],
  },
  bedrock: {
    label: 'AWS Bedrock',
    defaultBaseUrl: '',
    models: [],
  },
  mistral: {
    label: 'Mistral',
    defaultBaseUrl: 'https://api.mistral.ai/v1',
    models: [
      { modelId: 'mistral-large-latest', name: 'Mistral Large', contextWindow: 128000, isMultimodal: false, reasoning: false, maxTokens: 0 },
    ],
  },
  xai: {
    label: 'xAI',
    defaultBaseUrl: 'https://api.x.ai/v1',
    models: [
      { modelId: 'grok-3', name: 'Grok 3', contextWindow: 131072, isMultimodal: false, reasoning: false, maxTokens: 0 },
      { modelId: 'grok-3-mini', name: 'Grok 3 Mini', contextWindow: 131072, isMultimodal: false, reasoning: true, maxTokens: 0 },
    ],
  },
  ollama: {
    label: 'Ollama',
    defaultBaseUrl: 'http://localhost:11434/v1',
    models: [],
  },
  dashscope: {
    label: 'DashScope (通义千问)',
    defaultBaseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    models: [
      { modelId: 'qwen-max', name: 'Qwen Max', contextWindow: 32768, isMultimodal: false, reasoning: false, maxTokens: 0 },
      { modelId: 'qwen-plus', name: 'Qwen Plus', contextWindow: 131072, isMultimodal: false, reasoning: false, maxTokens: 0 },
      { modelId: 'qwen-turbo', name: 'Qwen Turbo', contextWindow: 131072, isMultimodal: false, reasoning: false, maxTokens: 0 },
    ],
  },
  deepseek: {
    label: 'DeepSeek',
    defaultBaseUrl: 'https://api.deepseek.com',
    models: [
      { modelId: 'deepseek-chat', name: 'DeepSeek V3', contextWindow: 65536, isMultimodal: false, reasoning: false, maxTokens: 8192 },
      { modelId: 'deepseek-reasoner', name: 'DeepSeek R1', contextWindow: 65536, isMultimodal: false, reasoning: true, maxTokens: 8192 },
    ],
  },
  'zai-global': {
    label: '字节智能 (Global)',
    defaultBaseUrl: '',
    models: [],
  },
  'zai-cn': {
    label: '字节智能 (CN)',
    defaultBaseUrl: '',
    models: [],
  },
  'zai-coding-global': {
    label: '字节智能 Coding (Global)',
    defaultBaseUrl: '',
    models: [],
  },
  'zai-coding-cn': {
    label: '字节智能 Coding (CN)',
    defaultBaseUrl: '',
    models: [],
  },
  'minimax-global': {
    label: 'MiniMax (Global)',
    defaultBaseUrl: '',
    models: [],
  },
  'minimax-cn': {
    label: 'MiniMax (CN)',
    defaultBaseUrl: '',
    models: [],
  },
  'moonshot-global': {
    label: 'Moonshot (Global)',
    defaultBaseUrl: '',
    models: [],
  },
  'moonshot-cn': {
    label: 'Moonshot (CN)',
    defaultBaseUrl: 'https://api.moonshot.cn/v1',
    models: [
      { modelId: 'moonshot-v1-128k', name: 'Moonshot v1 128K', contextWindow: 128000, isMultimodal: false, reasoning: false, maxTokens: 0 },
    ],
  },
  volcengine: {
    label: '火山引擎',
    defaultBaseUrl: '',
    models: [],
  },
  'volcengine-coding': {
    label: '火山引擎 Coding',
    defaultBaseUrl: '',
    models: [],
  },
  qianfan: {
    label: '百度千帆',
    defaultBaseUrl: '',
    models: [],
  },
  groq: {
    label: 'Groq',
    defaultBaseUrl: 'https://api.groq.com/openai/v1',
    models: [
      { modelId: 'llama-3.3-70b-versatile', name: 'Llama 3.3 70B', contextWindow: 128000, isMultimodal: false, reasoning: false, maxTokens: 0 },
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
      { modelId: 'sonar-pro', name: 'Sonar Pro', contextWindow: 200000, isMultimodal: false, reasoning: false, maxTokens: 0 },
      { modelId: 'sonar', name: 'Sonar', contextWindow: 128000, isMultimodal: false, reasoning: false, maxTokens: 0 },
    ],
  },
}

export function getProviderLabel(clientType: string): string {
  return providerCatalog[clientType]?.label ?? clientType
}

export function getProviderDefaultBaseUrl(clientType: string): string {
  return providerCatalog[clientType]?.defaultBaseUrl ?? ''
}

export function getProviderModels(clientType: string): CatalogModel[] {
  return providerCatalog[clientType]?.models ?? []
}
