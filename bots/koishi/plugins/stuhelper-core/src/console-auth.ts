const CONSOLE_ADMIN_PASSWORD_ENV = 'STUHELPER_CONSOLE_ADMIN_PASSWORD'
const PLACEHOLDER_PASSWORDS = new Set([
  'dev_console_admin_password_change_me',
])

export function validateConsoleAdminPassword(password: string | undefined) {
  if (password === undefined || password.trim().length === 0) {
    throw new Error(`${CONSOLE_ADMIN_PASSWORD_ENV} must be set to a non-empty string`)
  }

  if (PLACEHOLDER_PASSWORDS.has(password.trim())) {
    throw new Error(`${CONSOLE_ADMIN_PASSWORD_ENV} must not use a known placeholder password`)
  }

  return password
}
