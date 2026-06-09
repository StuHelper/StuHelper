import assert from 'node:assert/strict'
import test from 'node:test'

import { parseGroupGuardBehaviorSettingsInput } from './group-guard-behavior-settings-api'

test('parseGroupGuardBehaviorSettingsInput accepts fun runtime settings', () => {
  const input = parseGroupGuardBehaviorSettingsInput({
    fun: {
      diceSides: 20,
      muteLotteryBaseSeconds: 60,
      muteLotteryMaxSeconds: 300,
      muteLotteryPityThreshold: 2,
      muteLotteryPitySeconds: 120,
    },
  })

  assert.deepEqual(input, {
    fun: {
      diceSides: 20,
      muteLotteryBaseSeconds: 60,
      muteLotteryMaxSeconds: 300,
      muteLotteryPityThreshold: 2,
      muteLotteryPitySeconds: 120,
    },
  })
})

test('parseGroupGuardBehaviorSettingsInput accepts moderation runtime settings', () => {
  const input = parseGroupGuardBehaviorSettingsInput({
    moderation: {
      repeatThreshold: 4,
      repeatWindowSize: 5,
      warningThresholdExpression: ' warnings >= 1 ',
      defaultMuteSeconds: 180,
      antiRecallNotify: false,
    },
  })

  assert.deepEqual(input, {
    moderation: {
      repeatThreshold: 4,
      repeatWindowSize: 5,
      warningThresholdExpression: 'warnings >= 1',
      defaultMuteSeconds: 180,
      antiRecallNotify: false,
    },
  })
})

test('parseGroupGuardBehaviorSettingsInput rejects native-plugin-only and unknown fields', () => {
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      platform: { serviceToken: 'secret' },
    }),
    /unsupported field: platform/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      fun: {
        diceSides: 20,
        commandDescription: 'roll',
      },
    }),
    /unsupported field: commandDescription/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      fun: {
        diceSides: 0,
      },
    }),
    /diceSides must be a positive integer/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      fun: {
        diceSides: 1001,
      },
    }),
    /diceSides must be at most 1000/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        keywordRules: [],
      },
    }),
    /unsupported field: keywordRules/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        repeatThreshold: 0,
      },
    }),
    /repeatThreshold must be a positive integer/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        repeatWindowSize: 10001,
      },
    }),
    /repeatWindowSize must be at most 10000/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        warningThresholdExpression: '   ',
      },
    }),
    /warningThresholdExpression must be a non-empty string/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        defaultMuteSeconds: 2_592_001,
      },
    }),
    /defaultMuteSeconds must be at most 2592000/,
  )
  assert.throws(
    () => parseGroupGuardBehaviorSettingsInput({
      moderation: {
        antiRecallNotify: 'true',
      },
    }),
    /antiRecallNotify must be a boolean/,
  )
})
