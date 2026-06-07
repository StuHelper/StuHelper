import { describe, expect, it } from 'vitest'

import { readResourceDownloadURLPayload } from '../resourcePayload'

describe('resource payload readers', () => {
  it('accepts same-site relative and http download URLs', () => {
    expect(readResourceDownloadURLPayload({ url: '/resource-downloads/math.pdf' })).toEqual({
      url: '/resource-downloads/math.pdf',
    })
    expect(readResourceDownloadURLPayload({ url: 'https://files.example.com/math.pdf' })).toEqual({
      url: 'https://files.example.com/math.pdf',
    })
    expect(readResourceDownloadURLPayload({ url: 'http://127.0.0.1:9000/math.pdf' })).toEqual({
      url: 'http://127.0.0.1:9000/math.pdf',
    })
  })

  it('rejects unsafe download URL protocols', () => {
    for (const url of [
      'javascript:alert(1)',
      'data:text/html;base64,PGgxPkJvb208L2gxPg==',
      'ftp://files.example.com/math.pdf',
      '//files.example.com/math.pdf',
    ]) {
      expect(() => readResourceDownloadURLPayload({ url })).toThrow(
        'Invalid resource download response',
      )
    }
  })
})
