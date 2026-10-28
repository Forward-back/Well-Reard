import assert from 'node:assert/strict'
import test from 'node:test'

import { shouldRefreshGazeStatusAfterKnowledgePoll } from './gazeStatusRefresh.ts'

test('refreshes gaze status when a polled document leaves an in-flight state', () => {
  assert.equal(
    shouldRefreshGazeStatusAfterKnowledgePoll(
      { parse_status: 'finalizing', summary_status: 'processing' },
      { parse_status: 'completed', summary_status: 'completed' },
    ),
    true,
  )
})

test('does not refresh gaze status for ordinary in-flight polling updates', () => {
  assert.equal(
    shouldRefreshGazeStatusAfterKnowledgePoll(
      { parse_status: 'pending' },
      { parse_status: 'processing' },
    ),
    false,
  )
})
