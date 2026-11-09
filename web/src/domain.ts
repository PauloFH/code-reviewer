// Valores espelhados do backend (Go). Tipos e rótulos da interface derivam destas tabelas.

export const SCOPES = [
  { value: 'global', label: 'Global', hint: 'padrão para todos' },
  { value: 'client', label: 'Cliente', hint: 'host Git' },
  { value: 'project', label: 'Projeto', hint: 'repositório específico' },
] as const
export type Scope = (typeof SCOPES)[number]['value']

export const ROLES = [
  { value: 'super', label: 'Super', canManageResources: true, canManageUsers: true },
  { value: 'gestor', label: 'Gestor', canManageResources: true, canManageUsers: false },
  { value: 'padrao', label: 'Padrão', canManageResources: false, canManageUsers: false },
] as const
export type Role = (typeof ROLES)[number]['value']
export type Capability = 'canManageResources' | 'canManageUsers'

export const PLATFORMS = [
  { value: 'gitlab', label: 'GitLab', initials: 'GL' },
  { value: 'gitlab-selfhosted', label: 'GitLab auto-hospedado', initials: 'GL' },
  { value: 'github', label: 'GitHub', initials: 'GH' },
  { value: 'gitea', label: 'Gitea', initials: 'GT' },
  { value: 'gitea-selfhosted', label: 'Gitea auto-hospedado', initials: 'GT' },
] as const
export type Platform = (typeof PLATFORMS)[number]['value']

export const AI_MODES = [
  { value: 'api', label: 'API', hint: 'chave' },
  { value: 'account', label: 'Conta', hint: 'login' },
] as const
export type AIMode = (typeof AI_MODES)[number]['value']

// Situação nunca só por cor: cada uma tem sinal e rótulo.
export const EVENT_STATUSES = [
  { value: 'success', sign: '+', label: 'publicada', tone: 'ok' },
  { value: 'error', sign: '−', label: 'erro', tone: 'danger' },
  { value: 'skipped', sign: '~', label: 'ignorada', tone: 'warn' },
] as const
export type EventStatus = (typeof EVENT_STATUSES)[number]['value']

// Sem `need`, a página é de todo usuário logado. `key` é o atalho de uma tecla.
export const NAV = [
  { path: '/dashboard', label: 'Painel', key: 'p', section: 'Operação', need: undefined },
  { path: '/events', label: 'Revisões', key: 'r', section: 'Operação', need: undefined },
  { path: '/clients', label: 'Clientes', key: 'c', section: 'Operação', need: 'canManageResources' },
  { path: '/prompts', label: 'Prompts', key: 'o', section: 'Configuração', need: 'canManageResources' },
  { path: '/ai-configs', label: 'Modelos', key: 'm', section: 'Configuração', need: 'canManageResources' },
  { path: '/review-profiles', label: 'Perfis de revisão', key: 'f', section: 'Configuração', need: 'canManageResources' },
  { path: '/users', label: 'Usuários', key: 'u', section: 'Acesso', need: 'canManageUsers' },
] as const satisfies readonly { path: string; label: string; key: string; section: string; need?: Capability }[]
export type NavPath = (typeof NAV)[number]['path']

const byValue = <T extends { value: string }>(list: readonly T[], v: string) => list.find((x) => x.value === v)

export const scopeLabel = (s: string) => byValue(SCOPES, s)?.label ?? s
export const roleLabel = (r: string) => byValue(ROLES, r)?.label ?? r
export const aiModeLabel = (m: string) => byValue(AI_MODES, m)?.label ?? m
export const platformLabel = (p: string) => byValue(PLATFORMS, p)?.label ?? p
export const platformInitials = (p: string) => byValue(PLATFORMS, p)?.initials ?? p.slice(0, 2).toUpperCase()

// Papel desconhecido (usuário não carregado) não tem nenhuma capacidade.
export function roleCan(role: string | undefined, need: Capability | undefined): boolean {
  if (!need) return true
  return !!(role && byValue(ROLES, role)?.[need])
}

export function statusMark(status: string): { sign: string; label: string; tone: string } {
  return byValue(EVENT_STATUSES, status) ?? { sign: '~', label: status, tone: 'warn' }
}
