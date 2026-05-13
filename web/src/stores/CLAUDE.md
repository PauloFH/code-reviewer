# CLAUDE.md — web/src/stores/

## `auth.ts` — useAuthStore

Gerencia autenticação JWT no lado do cliente.

```ts
const auth = useAuthStore()
auth.token           // ref<string | null> — JWT; null = não autenticado
auth.isAuthenticated // computed<boolean>
auth.login(username, password)  // chama loginAPI, persiste token em localStorage
auth.logout()                   // limpa token e localStorage
```

**Persistência:** o token é lido do `localStorage` na inicialização (`localStorage.getItem('jwt')`). Isso mantém a sessão entre reloads.

**Fluxo de 401:** `http.ts` chama `auth.logout()` automaticamente em qualquer resposta 401, redirecionando para `/login`. Não é necessário checar 401 manualmente nas views.

---

## `toast.ts` — useToastStore

Notificações temporárias (toasts).

```ts
const toast = useToastStore()
toast.push(message, type?, ttl?)  // type: 'success' | 'error' | 'info'; ttl padrão: 3500ms
toast.remove(id)                  // removido automaticamente após ttl
toast.toasts                      // ref<Toast[]> — lido pelo ToastContainer
```

**Interface Toast:**
```ts
{ id: number; message: string; type: ToastType }
```

`push` atribui um id incremental, insere o toast na lista e agenda `remove(id)` após `ttl` ms. O `ToastContainer` observa `toasts` e renderiza cada um com cor por tipo.
