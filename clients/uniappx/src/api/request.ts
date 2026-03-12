type JsonValue = string | number | boolean | null | JsonObject | JsonArray
type JsonObject = { [key: string]: JsonValue }
type JsonArray = JsonValue[]

function request<T>(url: string, method: 'GET' | 'POST' | 'PUT' | 'DELETE', data?: JsonObject): Promise<T> {
  const token = uni.getStorageSync('token')

  return new Promise((resolve, reject) => {
    uni.request({
      url: `${import.meta.env.VITE_API_URL}${url}`,
      method,
      data,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? `Bearer ${token}` : ''
      },
      success: (res) => {
        if (res.statusCode === 200) {
          resolve(res.data as T)
        } else if (res.statusCode === 401) {
          uni.removeStorageSync('token')
          uni.navigateTo({ url: '/pages/auth/login' })
          reject(new Error('Unauthorized'))
        } else {
          reject(new Error('Request failed'))
        }
      },
      fail: reject
    })
  })
}

export const apiClient = {
  get: <T>(url: string, params?: Record<string, string>) => {
    const query = params ? '?' + new URLSearchParams(params).toString() : ''
    return request<T>(url + query, 'GET')
  },
  post: <T>(url: string, data?: JsonObject) => request<T>(url, 'POST', data),
  put: <T>(url: string, data?: JsonObject) => request<T>(url, 'PUT', data),
  delete: <T>(url: string) => request<T>(url, 'DELETE')
}
