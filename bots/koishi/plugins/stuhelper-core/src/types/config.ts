declare module 'koishi' {
  interface Config {
    keywords: string[]
    warnLimit: number
    banTimes: {
      expression: string
    }
    forbidden: {
      autoDelete: boolean
      autoBan: boolean
      autoKick: boolean
      muteDuration: number
      keywords: string[]
    }
    defaultWelcome: string
    banme: {
      enabled: boolean
      baseMin: number
      baseMax: number
      growthRate: number
      autoBan?: boolean
      jackpot: {
        enabled: boolean
        baseProb: number
        softPity: number
        hardPity: number
        upDuration: string
        loseDuration: string
      }
    }
    friendRequest: {
      enabled: boolean
      keywords: string[]
      rejectMessage: string
    }
    guildRequest: {
      enabled: boolean
      keywords: string[]
      rejectMessage: string
    }
    setTitle: {
      enabled: boolean
      authority: number
      maxLength: number
    }
    antiRepeat: {
      enabled: boolean
      threshold: number
    }
    openai: {
      enabled: boolean
      apiKey: string
      apiUrl: string
      maxTokens: number
      temperature: number
      model: string
      systemPrompt: string
      contextLimit: number
      translatePrompt: string
    }
    antiRecall: {
      enabled: boolean
      retentionDays: number
      maxRecordsPerUser: number
      showOriginalTime: boolean
      authority: number
    }
  }
}

export interface Config {
  keywords: string[]
  warnLimit: number
  banTimes: {
    expression: string
  }
  forbidden: {
    autoDelete: boolean
    autoBan: boolean
    autoKick: boolean
    muteDuration: number
    keywords: string[]
  }
  defaultWelcome?: string
  defaultGoodbye?: string
  dice: DiceConfig
  banme: BanMeConfig
  friendRequest: {
    enabled: boolean
    keywords: string[]
    rejectMessage: string
  }
  guildRequest: {
    enabled: boolean
    rejectMessage: string
  }
  setTitle: {
    enabled: boolean
    authority: number
    maxLength: number
  }
  antiRepeat: AntiRepeatConfig
  openai: {
    enabled: boolean
    chatEnabled?: boolean
    translateEnabled?: boolean
    apiKey: string
    apiUrl: string
    model: string
    systemPrompt: string
    translatePrompt: string
    maxTokens: number
    temperature: number
    contextLimit: number
  }
  report: {
    enabled: boolean
    authority: number
    autoProcess: boolean
    defaultPrompt: string
    contextPrompt: string
    maxReportTime: number
    guildConfigs: Record<string, {
      enabled: boolean
      includeContext: boolean
      contextSize: number
      autoProcess: boolean
    }>
    maxReportCooldown: number
    minAuthorityNoLimit: number
  }
  antiRecall: {
    enabled: boolean
    retentionDays: number
    maxRecordsPerUser: number
    showOriginalTime: boolean
  }
}

export interface ReportConfig {
  enabled: boolean
  authority: number
  autoProcess: boolean
  maxReportCooldown: number
  minAuthorityNoLimit: number
  maxReportTime: number
  defaultPrompt?: string
  contextPrompt?: string
  guildConfigs?: Record<string, ReportGuildConfig>
}

export interface ReportGuildConfig {
  enabled: boolean
  autoProcess?: boolean
  includeContext?: boolean
  contextSize?: number
}

export interface GroupConfig {
  keywords?: string[]
  approvalKeywords?: string[]
  auto?: string
  reject?: string
  forbidden?: {
    autoDelete: boolean
    autoBan: boolean
    autoKick: boolean
    muteDuration: number
    echo?: boolean
  }
  welcomeMsg?: string
  goodbyeMsg?: string
  welcomeEnabled?: boolean
  goodbyeEnabled?: boolean
  levelLimit?: number
  leaveCooldown?: number
  warnLimit?: number
  banme?: BanMeConfig
  dice?: DiceConfig
  antiRepeat?: AntiRepeatConfig
  openai?: {
    enabled: boolean
    chatEnabled?: boolean
    translateEnabled?: boolean
    systemPrompt?: string
    translatePrompt?: string
  }
  antiRecall?: {
    enabled: boolean
    retentionDays?: number
    maxRecordsPerUser?: number
  }
  report?: {
    enabled: boolean
    autoProcess?: boolean
    includeContext?: boolean
    contextSize?: number
  }
}

export interface DiceConfig {
  enabled?: boolean
  lengthLimit?: number
}

export interface AntiRepeatConfig {
  enabled: boolean
  threshold: number
}

export interface BanMeConfig {
  enabled: boolean
  baseMin: number
  baseMax: number
  growthRate: number
  autoBan?: boolean
  jackpot: {
    enabled: boolean
    baseProb: number
    softPity: number
    hardPity: number
    upDuration: string
    loseDuration: string
  }
}
