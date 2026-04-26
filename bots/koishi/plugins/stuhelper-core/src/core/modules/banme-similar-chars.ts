import * as fs from 'node:fs'

type ModuleLogger = {
  error(...args: unknown[]): void
}

const WHITESPACE_PATTERN = /[\u0000-\u0020\u00A0\u1680\u180E\u2000-\u200F\u2028-\u202F\u2060-\u206F\u205F\u3000\uFEFF]/g
const PUNCTUATION_PATTERN = /[.,\/#!$%\^&\*;:{}=\-_`~()]/g
const ZERO_WIDTH_PATTERN = /[\u200B-\u200D\uFEFF]/g
const VARIATION_PATTERN = /[\uE000-\uF8FF\uFE00-\uFE0F\uFE20-\uFE2F]/g
const COMBINING_PATTERN = /[\u0300-\u036F\u1AB0-\u1AFF\u20D0-\u20FF]/g

export type SimilarChars = Record<string, string>

export const DEFAULT_SIMILAR_CHARS: SimilarChars = {
  'α': 'a', 'а': 'a', 'Α': 'a', 'А': 'a', 'ɒ': 'a', 'ɐ': 'a', '𝐚': 'a', '𝐀': 'a', '₳': 'a', 'ₐ': 'a', 'ₔ': 'a', 'ₕ': 'a', '₠': 'a', '𝓪': 'a', '4': 'a',
  'е': 'e', 'Е': 'e', 'ε': 'e', 'Ε': 'e', 'ë': 'e', 'Ë': 'e', '𝐞': 'e', '𝐄': 'e', 'ə': 'e', 'Э': 'e', 'э': 'e', '𝓮': 'e',
  'м': 'm', 'М': 'm', '𝐦': 'm', '𝐌': 'm', 'rn': 'm', 'ₘ': 'm', '₞': 'm', '₥': 'm', '₩': 'm', '₼': 'm', 'ɱ': 'm', '𝓶': 'm',
  'н': 'n', 'Н': 'n', 'η': 'n', 'Ν': 'n', '𝐧': 'n', '𝐍': 'n', 'И': 'n', 'ん': 'n', 'ₙ': 'n', '₦': 'n', 'П': 'n', 'п': 'n', '∩': 'n', 'ñ': 'n', '𝓷': 'n',
  'в': 'b', 'В': 'b', 'Ь': 'b', 'ь': 'b', 'β': 'b', 'Β': 'B', '𝐛': 'b', '𝐁': 'B', '♭': 'b', 'ß': 'b', '₧': 'b', '₨': 'b', '₿': 'b', '𝓫': 'b',
  '我': 'me',
  '禁言': 'ban',
  '禁': 'ban',
  'mute': 'ban',
  'myself': 'me',
}

export class SimilarCharsStore {
  constructor(
    private readonly path: string,
    private readonly logger: ModuleLogger,
  ) {}

  ensure(): void {
    try {
      if (!fs.existsSync(this.path)) {
        this.writeDefaults()
      }
    } catch {
      this.writeDefaults()
    }
  }

  read(): SimilarChars | null {
    try {
      if (fs.existsSync(this.path)) {
        return JSON.parse(fs.readFileSync(this.path, 'utf-8'))
      }
    } catch (error) {
      this.logger.error(`[BanmeModule] 读取文件失败: ${this.path}`, error)
    }
    return null
  }

  save(data: SimilarChars): void {
    try {
      fs.writeFileSync(this.path, JSON.stringify(data, null, 2), 'utf-8')
    } catch (error) {
      this.logger.error(`[BanmeModule] 保存文件失败: ${this.path}`, error)
    }
  }

  writeDefaults(): void {
    this.save(DEFAULT_SIMILAR_CHARS)
  }
}

export function normalizeBanmeCommand(command: string, similarChars: SimilarChars): string {
  let normalized = command
    .replace(WHITESPACE_PATTERN, '')
    .replace(PUNCTUATION_PATTERN, '')
    .replace(ZERO_WIDTH_PATTERN, '')
    .replace(VARIATION_PATTERN, '')
    .replace(COMBINING_PATTERN, '')

  for (const [char, replacement] of Object.entries(similarChars)) {
    normalized = normalized.replace(new RegExp(char, 'g'), replacement)
  }

  return normalized
    .replace(PUNCTUATION_PATTERN, '')
    .replace(/(.)\1+/g, '$1')
    .toLowerCase()
}
