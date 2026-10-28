import { diffGazeLines, type GazeDiffLine } from './gazeLineDiff'

export type GazeRevisionDiffField = 'title' | 'summary' | 'content'

export interface GazeRevisionDiffSection {
  field: GazeRevisionDiffField
  lines: GazeDiffLine[]
}

export interface GazeRevisionSnapshot {
  title: string
  summary: string
  content: string
}

function hasChanges(lines: GazeDiffLine[]): boolean {
  return lines.some((line) => line.type !== 'same')
}

/** Compare a stored revision against the current page across title, summary, and body. */
export function diffGazeRevision(
  oldSnap: GazeRevisionSnapshot,
  newSnap: GazeRevisionSnapshot,
): GazeRevisionDiffSection[] {
  const sections: GazeRevisionDiffSection[] = []

  const titleLines = diffGazeLines(oldSnap.title, newSnap.title)
  if (hasChanges(titleLines)) {
    sections.push({ field: 'title', lines: titleLines })
  }

  const summaryLines = diffGazeLines(oldSnap.summary, newSnap.summary)
  if (hasChanges(summaryLines)) {
    sections.push({ field: 'summary', lines: summaryLines })
  }

  const contentLines = diffGazeLines(oldSnap.content, newSnap.content)
  if (hasChanges(contentLines)) {
    sections.push({ field: 'content', lines: contentLines })
  }

  return sections
}
