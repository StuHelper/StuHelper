const CONSOLE_ADMIN_PASSWORD_ENV = 'STUHELPER_CONSOLE_ADMIN_PASSWORD'

export function validateConsoleAdminPassword(password: string | undefined) {
  if (password === undefined || password.trim().length === 0) {
    throw new Error(`${CONSOLE_ADMIN_PASSWORD_ENV} must be set to a non-empty string`)
  }

  return password
}
