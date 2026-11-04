export const NAV_SHORTCUTS: Record<string, string> = {
  p: '/dashboard',
  r: '/events',
  c: '/clients',
  o: '/prompts',
  m: '/ai-configs',
  f: '/review-profiles',
  u: '/users',
}

// Inclui contenteditable: o editor Tiptap não é input, e digitar "p"/"o" não pode trocar de tela.
export function isEditableTarget(target: { tagName?: string; isContentEditable?: boolean } | null): boolean {
  if (!target) return false
  return ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName ?? '') || target.isContentEditable === true
}
