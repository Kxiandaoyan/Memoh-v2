export enum ClientType {
  OpenAI = 'openai',
  OpenAICompat = 'openai-compat',
  Anthropic = 'anthropic',
  Google = 'google',
  Azure = 'azure',
  Bedrock = 'bedrock',
  Mistral = 'mistral',
  XAI = 'xai',
  Ollama = 'ollama',
  Dashscope = 'dashscope',
  DeepSeek = 'deepseek',
  ZaiGlobal = 'zai-global',
  ZaiCN = 'zai-cn',
  ZaiCodingGlobal = 'zai-coding-global',
  ZaiCodingCN = 'zai-coding-cn',
  MinimaxGlobal = 'minimax-global',
  MinimaxCN = 'minimax-cn',
  MoonshotGlobal = 'moonshot-global',
  MoonshotCN = 'moonshot-cn',
  Volcengine = 'volcengine',
  VolcengineCoding = 'volcengine-coding',
  Qianfan = 'qianfan',
  Groq = 'groq',
  OpenRouter = 'openrouter',
  Together = 'together',
  Fireworks = 'fireworks',
  Perplexity = 'perplexity',
  Zhipu = 'zhipu',
  Siliconflow = 'siliconflow',
  Nvidia = 'nvidia',
  Bailing = 'bailing',
  Xiaomi = 'xiaomi',
  Longcat = 'longcat',
  ModelScope = 'modelscope',
}

export enum ModelInput {
  Text = 'text',
  Image = 'image',
}

/** Providers that natively support role:"system" in the messages array. */
export const SYSTEM_SAFE_PROVIDERS = new Set<string>([
  'openai', 'anthropic', 'google', 'azure', 'bedrock', 'mistral', 'xai',
  'deepseek', 'groq', 'openrouter', 'together', 'fireworks', 'perplexity',
  'zhipu', 'siliconflow', 'nvidia', 'bailing', 'xiaomi', 'longcat', 'modelscope',
])

export interface ModelConfig {
  apiKey: string
  baseUrl: string
  modelId: string
  clientType: ClientType
  input: ModelInput[]
  reasoning?: boolean
  maxTokens?: number
}