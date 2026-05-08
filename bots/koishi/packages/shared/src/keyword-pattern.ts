const DEFAULT_MAX_REGEX_PATTERN_LENGTH = 256
const BACKREFERENCE_PATTERN = /\\(?:[1-9]|k<[^>]+>)/
const NESTED_QUANTIFIER_PATTERN = /\((?=[^)]*(?:[+*]|\{\d+(?:,\d*)?\}))[^)]*\)(?:[+*]|\{\d+(?:,\d*)?\})/

export class UnsafeKeywordPatternError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'UnsafeKeywordPatternError'
  }
}

export function testSafeKeywordRegex(pattern: string, content: string): boolean {
  assertSafeKeywordRegex(pattern)
  return new RegExp(pattern, 'i').test(content)
}

export function assertSafeKeywordRegex(
  pattern: string,
  maxLength = DEFAULT_MAX_REGEX_PATTERN_LENGTH,
): void {
  const source = pattern.trim()
  if (!source) throw new UnsafeKeywordPatternError('unsafe keyword regex: pattern is empty')
  if (source.length > maxLength) {
    throw new UnsafeKeywordPatternError(`unsafe keyword regex: pattern exceeds ${maxLength} characters`)
  }
  if (BACKREFERENCE_PATTERN.test(source)) {
    throw new UnsafeKeywordPatternError('unsafe keyword regex: backreferences are not allowed')
  }
  if (NESTED_QUANTIFIER_PATTERN.test(stripEscapedRegexTokens(source))) {
    throw new UnsafeKeywordPatternError('unsafe keyword regex: nested quantifiers are not allowed')
  }
  assertCompiles(source)
}

function assertCompiles(pattern: string): void {
  try {
    new RegExp(pattern, 'i')
  } catch (cause) {
    const message = cause instanceof Error ? cause.message : String(cause)
    throw new UnsafeKeywordPatternError(`unsafe keyword regex: ${message}`)
  }
}

function stripEscapedRegexTokens(pattern: string): string {
  return pattern
    .replace(/\\./g, '')
    .replace(/\[[^\]]*]/g, '')
}
