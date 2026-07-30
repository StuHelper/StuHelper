// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const gsapMocks = vi.hoisted(() => ({
  to: vi.fn(),
  killTweensOf: vi.fn(),
}))

vi.mock('gsap', () => ({
  default: gsapMocks,
}))

Object.defineProperty(window, 'matchMedia', {
  configurable: true,
  value: vi.fn().mockReturnValue({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }),
})

const { default: ParticleBackground } =
  await import('../ParticleBackground.vue')

let frameCallbacks = new Map<number, FrameRequestCallback>()
let nextFrameID = 0
let requestFrameMock = vi.fn()
let cancelFrameMock = vi.fn()
let restoreCanvasContext = () => undefined

function runFrame(id: number) {
  const callback = frameCallbacks.get(id)
  if (!callback) throw new Error(`missing animation frame ${id}`)
  frameCallbacks.delete(id)
  callback(0)
}

describe('ParticleBackground tween lifecycle', () => {
  beforeEach(() => {
    frameCallbacks = new Map()
    nextFrameID = 0
    requestFrameMock = vi.fn((callback: FrameRequestCallback) => {
      const id = ++nextFrameID
      frameCallbacks.set(id, callback)
      return id
    })
    cancelFrameMock = vi.fn((id: number) => {
      frameCallbacks.delete(id)
    })
    vi.stubGlobal('requestAnimationFrame', requestFrameMock)
    vi.stubGlobal('cancelAnimationFrame', cancelFrameMock)

    const canvasContext = {
      clearRect: vi.fn(),
      beginPath: vi.fn(),
      arc: vi.fn(),
      fill: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      stroke: vi.fn(),
      fillStyle: '',
      strokeStyle: '',
      lineWidth: 0,
    } as unknown as CanvasRenderingContext2D
    const contextSpy = vi
      .spyOn(HTMLCanvasElement.prototype, 'getContext')
      .mockReturnValue(canvasContext)
    restoreCanvasContext = () => contextSpy.mockRestore()

    gsapMocks.to.mockReset()
    gsapMocks.killTweensOf.mockReset()
  })

  afterEach(() => {
    restoreCanvasContext()
    vi.unstubAllGlobals()
  })

  it('kills the current tween targets before replacing them on resize', () => {
    const wrapper = mount(ParticleBackground, {
      props: {
        particleCount: 3,
      },
    })

    const initialTargets = gsapMocks.to.mock.calls.map(([target]) => target)
    expect(initialTargets).toHaveLength(3)
    expect(gsapMocks.killTweensOf).toHaveBeenCalledWith([])

    gsapMocks.to.mockClear()
    gsapMocks.killTweensOf.mockClear()

    const frameCallsBeforeResize = requestFrameMock.mock.calls.length
    window.dispatchEvent(new Event('resize'))
    window.dispatchEvent(new Event('resize'))

    expect(requestFrameMock).toHaveBeenCalledTimes(frameCallsBeforeResize + 1)
    const resizeFrameID = nextFrameID
    runFrame(resizeFrameID)

    expect(gsapMocks.killTweensOf).toHaveBeenCalledTimes(1)
    expect(gsapMocks.killTweensOf).toHaveBeenCalledWith(initialTargets)
    expect(gsapMocks.killTweensOf.mock.invocationCallOrder[0]).toBeLessThan(
      gsapMocks.to.mock.invocationCallOrder[0],
    )

    const resizedTargets = gsapMocks.to.mock.calls.map(([target]) => target)
    expect(resizedTargets).toHaveLength(3)
    expect(
      resizedTargets.every(target => !initialTargets.includes(target)),
    ).toBe(true)

    gsapMocks.killTweensOf.mockClear()
    wrapper.unmount()

    expect(gsapMocks.killTweensOf).toHaveBeenCalledTimes(1)
    expect(gsapMocks.killTweensOf).toHaveBeenCalledWith(resizedTargets)
  })
})
