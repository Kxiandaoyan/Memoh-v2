import { createGateway as createAiGateway } from 'ai'
import { createOpenAI } from '@ai-sdk/openai'
import { createAnthropic } from '@ai-sdk/anthropic'
import { createGoogleGenerativeAI } from '@ai-sdk/google'
import { createAzure } from '@ai-sdk/azure'
import { createAmazonBedrock } from '@ai-sdk/amazon-bedrock'
import { createMistral } from '@ai-sdk/mistral'
import { createXai } from '@ai-sdk/xai'
import { ClientType, ModelConfig } from './types'

/**
 * Build providerOptions for generateText when reasoning=true.
 * Pass the returned object as `providerOptions` to generateText/streamText.
 */
export const getProviderOptions = (config: ModelConfig) => {
  if (!config.reasoning) return undefined
  return { openai: { reasoningEffort: 'high' } } as const
}

export const createModel = (model: ModelConfig) => {
  const apiKey = model.apiKey.trim()
  const baseURL = model.baseUrl.trim()
  const modelId = model.modelId.trim()

  switch (model.clientType) {
    case ClientType.OpenAI:
    case ClientType.OpenAICompat:
    case ClientType.Ollama:
    case ClientType.Dashscope: {
      // Force .chat() (Chat Completions API) for all OpenAI-based providers.
      // The default auto-detect may use the Responses API which produces
      // item_reference items that cause errors on subsequent turns.
      const provider = createOpenAI({ apiKey, baseURL })
      return provider.chat(modelId)
    }
    case ClientType.Anthropic:
      return createAnthropic({ apiKey, baseURL })(modelId)
    case ClientType.Google:
      return createGoogleGenerativeAI({ apiKey, baseURL })(modelId)
    case ClientType.Azure:
      return createAzure({ apiKey, baseURL })(modelId)
    case ClientType.Bedrock: {
      // Bedrock uses AWS credentials; apiKey as accessKeyId, metadata for secretAccessKey
      // Falls back to AWS default credential chain if not provided
      const opts: Record<string, string> = {}
      if (baseURL) opts.region = baseURL
      if (apiKey) opts.accessKeyId = apiKey
      return createAmazonBedrock(opts)(modelId)
    }
    case ClientType.Mistral:
      return createMistral({ apiKey, baseURL: baseURL || undefined })(modelId)
    case ClientType.XAI:
      return createXai({ apiKey, baseURL: baseURL || undefined })(modelId)
    case ClientType.DeepSeek:
    case ClientType.ZaiGlobal:
    case ClientType.ZaiCN:
    case ClientType.ZaiCodingGlobal:
    case ClientType.ZaiCodingCN:
    case ClientType.MinimaxGlobal:
    case ClientType.MinimaxCN:
    case ClientType.MoonshotGlobal:
    case ClientType.MoonshotCN:
    case ClientType.Volcengine:
    case ClientType.VolcengineCoding:
    case ClientType.Qianfan:
    case ClientType.Groq:
    case ClientType.OpenRouter:
    case ClientType.Together:
    case ClientType.Fireworks:
    case ClientType.Perplexity:
    case ClientType.Zhipu:
    case ClientType.Siliconflow:
    case ClientType.Nvidia:
    case ClientType.Bailing:
    case ClientType.Xiaomi:
    case ClientType.Longcat:
    case ClientType.ModelScope: {
      const provider = createOpenAI({ apiKey, baseURL })
      return provider.chat(modelId)
    }
    default:
      return createAiGateway({ apiKey, baseURL })(modelId)
  }
}