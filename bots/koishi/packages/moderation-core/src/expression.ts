import type { ThresholdMetrics } from './types'

interface Token {
  type: 'identifier' | 'number' | 'operator' | 'paren'
  value: string
}

interface State {
  index: number
  tokens: Token[]
}

const OPERATORS = ['>=', '<=', '==', '!=', '&&', '||', '>', '<']

export function evaluateThresholdExpression(input: string, metrics: ThresholdMetrics) {
  const state: State = { index: 0, tokens: tokenizeExpression(input) }
  const value = parseOr(state, metrics)
  if (state.index !== state.tokens.length) {
    throw new Error(`unexpected token: ${state.tokens[state.index]?.value}`)
  }
  return value
}

function tokenizeExpression(input: string) {
  const tokens: Token[] = []
  let index = 0
  while (index < input.length) {
    const char = input[index]
    if (/\s/.test(char)) {
      index += 1
      continue
    }
    const operator = OPERATORS.find((item) => input.startsWith(item, index))
    if (operator) {
      tokens.push({ type: 'operator', value: operator })
      index += operator.length
      continue
    }
    if (char === '(' || char === ')') {
      tokens.push({ type: 'paren', value: char })
      index += 1
      continue
    }
    const numberMatch = input.slice(index).match(/^\d+/)
    if (numberMatch) {
      tokens.push({ type: 'number', value: numberMatch[0] })
      index += numberMatch[0].length
      continue
    }
    const identifierMatch = input.slice(index).match(/^[a-zA-Z_][a-zA-Z0-9_]*/)
    if (identifierMatch) {
      tokens.push({ type: 'identifier', value: identifierMatch[0] })
      index += identifierMatch[0].length
      continue
    }
    throw new Error(`invalid expression near: ${input.slice(index)}`)
  }
  return tokens
}

function parseOr(state: State, metrics: ThresholdMetrics): boolean {
  let value = parseAnd(state, metrics)
  while (matchOperator(state, '||')) {
    const right = parseAnd(state, metrics)
    value = value || right
  }
  return value
}

function parseAnd(state: State, metrics: ThresholdMetrics): boolean {
  let value = parsePrimary(state, metrics)
  while (matchOperator(state, '&&')) {
    const right = parsePrimary(state, metrics)
    value = value && right
  }
  return value
}

function parsePrimary(state: State, metrics: ThresholdMetrics): boolean {
  if (matchParen(state, '(')) {
    const value = parseOr(state, metrics)
    expectParen(state, ')')
    return value
  }
  return parseComparison(state, metrics)
}

function parseComparison(state: State, metrics: ThresholdMetrics): boolean {
  const left = parseOperand(state, metrics)
  const operator = nextToken(state)
  if (!operator || operator.type !== 'operator' || ['&&', '||'].includes(operator.value)) {
    throw new Error('comparison operator is required')
  }
  const right = parseOperand(state, metrics)
  return compareValues(left, right, operator.value)
}

function parseOperand(state: State, metrics: ThresholdMetrics): number {
  const token = nextToken(state)
  if (!token) {
    throw new Error('unexpected end of expression')
  }
  if (token.type === 'number') {
    return Number(token.value)
  }
  if (token.type === 'identifier') {
    return metrics[token.value as keyof ThresholdMetrics] ?? 0
  }
  throw new Error(`unexpected operand token: ${token.value}`)
}

function compareValues(left: number, right: number, operator: string) {
  if (operator === '>=') return left >= right
  if (operator === '<=') return left <= right
  if (operator === '==') return left === right
  if (operator === '!=') return left !== right
  if (operator === '>') return left > right
  if (operator === '<') return left < right
  throw new Error(`unsupported operator: ${operator}`)
}

function matchOperator(state: State, operator: string) {
  const token = state.tokens[state.index]
  if (token?.type === 'operator' && token.value === operator) {
    state.index += 1
    return true
  }
  return false
}

function matchParen(state: State, paren: '(' | ')') {
  const token = state.tokens[state.index]
  if (token?.type === 'paren' && token.value === paren) {
    state.index += 1
    return true
  }
  return false
}

function expectParen(state: State, paren: '(' | ')') {
  if (!matchParen(state, paren)) {
    throw new Error(`expected ${paren}`)
  }
}

function nextToken(state: State) {
  const token = state.tokens[state.index]
  state.index += 1
  return token
}
