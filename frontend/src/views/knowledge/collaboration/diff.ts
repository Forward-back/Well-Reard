export type DiffRowType = 'ctx' | 'add' | 'del'

export interface DiffRow {
  type: DiffRowType
  text: string
  oldNo?: number
  newNo?: number
}

export interface DiffResult {
  rows: DiffRow[]
  added: number
  removed: number
  similarity: number
}

function splitLines(text: string): string[] {
  return String(text ?? '').replace(/\r\n/g, '\n').split('\n')
}

/** Classic LCS line diff. Documents are chunk-sized, so O(n*m) is fine. */
export function diffLines(before: string, after: string): DiffResult {
  const a = splitLines(before)
  const b = splitLines(after)
  const n = a.length
  const m = b.length
  const dp: Uint32Array[] = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = a[i] === b[j]
        ? dp[i + 1][j + 1] + 1
        : Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const rows: DiffRow[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      rows.push({ type: 'ctx', text: a[i], oldNo: i + 1, newNo: j + 1 })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      rows.push({ type: 'del', text: a[i], oldNo: i + 1 })
      i++
    } else {
      rows.push({ type: 'add', text: b[j], newNo: j + 1 })
      j++
    }
  }
  while (i < n) {
    rows.push({ type: 'del', text: a[i], oldNo: i + 1 })
    i++
  }
  while (j < m) {
    rows.push({ type: 'add', text: b[j], newNo: j + 1 })
    j++
  }
  const lcs = dp[0][0]
  const maxLen = Math.max(n, m) || 1
  const added = rows.filter((row) => row.type === 'add').length
  const removed = rows.filter((row) => row.type === 'del').length
  return {
    rows,
    added,
    removed,
    similarity: Math.round((lcs / maxLen) * 100),
  }
}

export function diffStats(before: string, after: string) {
  const result = diffLines(before, after)
  return { added: result.added, removed: result.removed, similarity: result.similarity }
}

export function diffMarkdownVersions(
  beforeChunks: Array<{ chunk_id: string; chunk_index: number; content: string }>,
  afterChunks: Array<{ chunk_id: string; chunk_index: number; content: string }>,
) {
  const afterMap = new Map(afterChunks.map((chunk) => [chunk.chunk_id, chunk]))
  const beforeMap = new Map(beforeChunks.map((chunk) => [chunk.chunk_id, chunk]))
  const chunkIds = new Set([...beforeMap.keys(), ...afterMap.keys()])
  const byIndex = (id: string) => beforeMap.get(id)?.chunk_index ?? afterMap.get(id)?.chunk_index ?? 0
  return [...chunkIds]
    .sort((left, right) => byIndex(left) - byIndex(right))
    .map((chunkId) => {
      const before = beforeMap.get(chunkId)
      const after = afterMap.get(chunkId)
      const result = diffLines(before?.content || '', after?.content || '')
      return {
        chunkId,
        chunkIndex: byIndex(chunkId),
        before: before?.content || '',
        after: after?.content || '',
        ...result,
      }
    })
    .filter((item) => item.added > 0 || item.removed > 0)
}
