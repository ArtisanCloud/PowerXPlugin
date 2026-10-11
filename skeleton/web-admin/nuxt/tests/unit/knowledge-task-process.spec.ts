import {describe,it,expect} from 'vitest'
import {hostTaskItem,taskProgress,taskSummary,taskPhase,taskDurationMs} from '../../app/composables/knowledge/taskProcess'

describe('knowledge task process presentation',()=>{
 it('distinguishes acceptance from execution and counts terminal failures without claiming success',()=>{
  const items=[{id:'1',title:'fixture',phase:taskPhase('queued')},{id:'2',title:'fixture',phase:taskPhase('running')},{id:'3',title:'fixture',phase:taskPhase('succeeded')},{id:'4',title:'fixture',phase:taskPhase('failed')}]
  expect(taskSummary(items)).toMatchObject({total:4,waiting:1,running:1,succeeded:1,failed:1,ended:2,endedPercent:50})
  expect(taskProgress(items[0])).toBe(0);expect(taskProgress(items[1])).toBeNull();expect(taskProgress(items[2])).toBe(100)
 })
 it('uses only valid reported execution percentages and does not convert unknown states into success',()=>{
  const running={id:'1',title:'fixture',phase:taskPhase('running')}
  expect(taskProgress({...running,progress:34})).toBe(34)
  expect(taskProgress({...running,progress:101})).toBeNull()
  expect(taskProgress({...running,progress:Number.NaN})).toBeNull()
  expect(taskSummary([{...running,phase:taskPhase('unpublished-state')}])).toMatchObject({succeeded:0,unknown:1,endedPercent:0})
 })
 it('keeps Local completion and Core completion semantics in the same view model',()=>{
  expect(taskPhase('completed')).toBe('succeeded');expect(taskPhase('succeeded')).toBe('succeeded')
  expect(taskPhase('accepted')).toBe('accepted');expect(taskPhase('processing')).toBe('running')
 })
 it('shows server execution duration only after a terminal state and preserves unavailable chunk counts',()=>{
  const start='2026-10-08T10:37:35.521Z',finish='2026-10-08T10:37:35.533Z'
  expect(hostTaskItem({job_id:'fixture',status:'succeeded',started_at:start,finished_at:finish,chunk_count:2},'fixture')).toMatchObject({phase:'succeeded',durationMs:12,chunks:2})
  const queued=hostTaskItem({job_id:'fixture',status:'queued',started_at:'0001-01-01T00:00:00Z',finished_at:'0001-01-01T00:00:00Z'},'fixture')
  expect(queued.durationMs).toBeUndefined();expect(queued.chunks).toBeUndefined()
  expect(taskDurationMs(start,undefined)).toBeUndefined();expect(taskDurationMs(finish,start)).toBeUndefined();expect(taskDurationMs('invalid',finish)).toBeUndefined()
 })
})
