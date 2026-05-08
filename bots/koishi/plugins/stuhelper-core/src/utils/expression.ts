const MAX_EXPRESSION_LOOPS = 999
const ALLOWED_EXPRESSION_PATTERN = /^[\d+\-*/()^.esqrtx]+$/
const SIMPLE_NUMBER_PATTERN = /^-?\d+\.?\d*$/
const POWER_PATTERN = /(-?\d+\.?\d*)\^(-?\d+\.?\d*)/
const MULTIPLY_DIVIDE_PATTERN = /(-?\d+\.?\d*)[*/](-?\d+\.?\d*)/
const SQRT_PATTERN = /sqrt\(([^()]+)\)/

export function evaluateExpression(expr: string): number {
  const normalized = normalizeExpression(expr)
  const withScientificNotation = expandScientificNotation(normalized)
  return calculateBasic(expandSquareRoots(withScientificNotation))
}

export function calculateBasic(expr: string): number {
  const normalized = normalizeSignRuns(expr)
  const withoutParentheses = reduceParentheses(normalized)
  const withoutPowers = reduceBinaryOperator({
    expr: withoutParentheses,
    marker: '^',
    pattern: POWER_PATTERN,
    calculate: (base, exp) => {
      return Math.pow(base, exp)
    },
  })
  const withoutMultiplication = reduceBinaryOperator({
    expr: withoutPowers,
    marker: '*/',
    pattern: MULTIPLY_DIVIDE_PATTERN,
    calculate: (left, right, match) => {
      return match.includes('*') ? left * right : left / right
    },
  })
  if (SIMPLE_NUMBER_PATTERN.test(withoutMultiplication)) return Number(withoutMultiplication)
  return sumAdditiveParts(parseAdditiveParts(withoutMultiplication))
}

function normalizeExpression(expr: string): string {
  const normalized = expr.replace(/\s/g, '').replace(/x/g, '*')
  if (!ALLOWED_EXPRESSION_PATTERN.test(normalized)) {
    throw new Error(`表达式包含非法字符: ${normalized}`)
  }
  assertBalancedParentheses(normalized)
  return normalized
}

function assertBalancedParentheses(expr: string): void {
  const openBrackets = (expr.match(/\(/g) || []).length
  const closeBrackets = (expr.match(/\)/g) || []).length
  if (openBrackets !== closeBrackets) throw new Error('表达式括号不匹配')
}

function expandScientificNotation(expr: string): string {
  return expr.replace(/(\d+)e(\d+)/g, (_, base, exp) => {
    return String(Number(base) * Math.pow(10, Number(exp)))
  })
}

function expandSquareRoots(expr: string): string {
  return reduceWhile(expr, 'sqrt', (current) => {
    const next = current.replace(SQRT_PATTERN, (_, num) => String(Math.sqrt(calculateBasic(num))))
    if (next === current) throw new Error(`表达式包含无法解析的 sqrt: ${current}`)
    return next
  })
}

function normalizeSignRuns(expr: string): string {
  return expr
    .replace(/\+-+/g, (match) => match.length % 2 === 0 ? '+' : '-')
    .replace(/--+/g, (match) => match.length % 2 === 0 ? '+' : '-')
}

function reduceParentheses(expr: string): string {
  return reduceWhile(expr, '(', (current) => {
    const next = current.replace(/\(([^()]+)\)/g, (_, subExpr) => String(calculateBasic(subExpr)))
    if (next === current) throw new Error(`表达式括号不匹配: ${current}`)
    return next
  })
}

function reduceBinaryOperator(input: {
  readonly expr: string
  readonly marker: string
  readonly pattern: RegExp
  readonly calculate: (left: number, right: number, match: string) => number
}): string {
  const { expr, marker, pattern, calculate } = input
  return reduceWhile(expr, marker, (current) => {
    const next = current.replace(pattern, (match, left, right) => {
      return String(calculate(Number(left), Number(right), match))
    })
    if (next === current) throw new Error(`表达式无法解析: ${current}`)
    return next
  })
}

function reduceWhile(expr: string, marker: string, reduce: (current: string) => string): string {
  let current = expr
  let loopCount = 0
  while (includesMarker(current, marker)) {
    if (++loopCount > MAX_EXPRESSION_LOOPS) throw new Error('表达式过于复杂，计算超时')
    current = reduce(current)
  }
  return current
}

function includesMarker(expr: string, marker: string): boolean {
  return marker === '*/' ? /[*/]/.test(expr) : expr.includes(marker)
}

function parseAdditiveParts(expr: string): Array<number | string> {
  const parts: Array<number | string> = []
  let currentNumber = ''
  let currentIsNegative = false

  for (let i = 0; i < expr.length; i++) {
    const char = expr[i]
    if (char === '-' && isUnaryMinus(expr, i)) {
      currentIsNegative = true
      continue
    }
    if (char === '+' || char === '-') {
      pushCurrentNumber(parts, currentNumber, currentIsNegative)
      parts.push(char)
      currentNumber = ''
      currentIsNegative = false
    } else if (/[\d.]/.test(char)) {
      currentNumber += char
    } else {
      throw new Error(`表达式包含非法字符: ${char}`)
    }
  }

  if (currentNumber !== '') pushCurrentNumber(parts, currentNumber, currentIsNegative)
  return parts
}

function isUnaryMinus(expr: string, index: number): boolean {
  return index === 0 || ['+', '-', '*', '/', '^'].includes(expr[index - 1])
}

function pushCurrentNumber(parts: Array<number | string>, value: string, isNegative: boolean): void {
  parts.push(isNegative ? -Number(value || '0') : Number(value || '0'))
}

function sumAdditiveParts(parts: Array<number | string>): number {
  let result = Number(parts[0])
  for (let i = 1; i < parts.length; i += 2) {
    const operator = parts[i]
    const operand = Number(parts[i + 1])
    if (operator === '+') result += operand
    if (operator === '-') result -= operand
  }
  return result
}
