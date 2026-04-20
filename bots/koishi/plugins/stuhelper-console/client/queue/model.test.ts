import assert from 'node:assert/strict'
import test from 'node:test'

import { getNextFocusableId } from './model'

test('getNextFocusableId returns the next row when later rows remain', () => {
  const nextId = getNextFocusableId({
    ids: ['rv_1', 'rv_2', 'rv_3'],
    currentId: 'rv_2',
    removedId: 'rv_2',
  })

  assert.equal(nextId, 'rv_3')
})

test('getNextFocusableId falls back to the previous row when current row is the tail', () => {
  const nextId = getNextFocusableId({
    ids: ['rv_1', 'rv_2', 'rv_3'],
    currentId: 'rv_3',
    removedId: 'rv_3',
  })

  assert.equal(nextId, 'rv_2')
})

test('getNextFocusableId uses the first remaining row when current id is missing', () => {
  const nextId = getNextFocusableId({
    ids: ['rv_1', 'rv_2', 'rv_3'],
    currentId: 'rv_9',
    removedId: 'rv_2',
  })

  assert.equal(nextId, 'rv_1')
})

test('getNextFocusableId returns empty string when queue becomes empty', () => {
  const nextId = getNextFocusableId({
    ids: ['rv_1'],
    currentId: 'rv_1',
    removedId: 'rv_1',
  })

  assert.equal(nextId, '')
})

test('getNextFocusableId ignores removedId when no row is removed yet', () => {
  const nextId = getNextFocusableId({
    ids: ['rv_1', 'rv_2', 'rv_3'],
    currentId: 'rv_2',
  })

  assert.equal(nextId, 'rv_2')
})
