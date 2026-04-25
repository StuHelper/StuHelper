export const DEFAULT_HTTP_STATUS_ERROR_CODES = {
  400: 'A0000400',
  401: 'A0010100',
  403: 'A0010200',
  404: 'A0000404',
  409: 'A0000409',
  422: 'A0000422',
  429: 'A0000429',
  500: 'B0000001',
  502: 'C0000001',
  503: 'B0000004',
  504: 'B0000006',
} as const satisfies Record<number, string>

export const DEFAULT_INTERNAL_ERROR_CODE = 'B0000001'

export function defaultHttpStatusErrorCode(status: number): string {
  return DEFAULT_HTTP_STATUS_ERROR_CODES[status as keyof typeof DEFAULT_HTTP_STATUS_ERROR_CODES]
    ?? DEFAULT_INTERNAL_ERROR_CODE
}
