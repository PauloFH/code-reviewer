// localStorage pode lançar (armazenamento bloqueado): nesse caso o valor vale só nesta sessão.
export function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    // sem persistência
  }
}

export function removeStorage(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    // sem persistência
  }
}

export function readSessionFlag(key: string): boolean {
  try {
    return sessionStorage.getItem(key) !== null
  } catch {
    return false
  }
}

export function writeSessionFlag(key: string): void {
  try {
    sessionStorage.setItem(key, '1')
  } catch {
    // sem persistência
  }
}
