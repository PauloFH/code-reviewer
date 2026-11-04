// O backend publica no SSE com id 0 (o id do banco não é preenchido); sem id local, o dedupe descartaria todo evento após o primeiro.
export function createEventAdmitter() {
  let local = 0
  return (feed: { id: number }[], e: { id: number }) => {
    if (!e.id) {
      e.id = --local
      return true
    }
    return !feed.some((x) => x.id === e.id)
  }
}
