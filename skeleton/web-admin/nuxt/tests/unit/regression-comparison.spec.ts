import { describe, expect, it } from 'vitest';
import type { AssertionResult } from '@artisan-cloud/plugin-framework-client';
import { regressionComparison } from '../../../../../framework/frontend/nuxt/framework-admin/layer/app/composables/regressionComparison';
import { regressionResultPresentations } from '../../app/composables/regressionPresentation';
import en from '../../i18n/locales/en.json';
const t = (key: string, params: Record<string, string | number> = {}) => {
  const value = key.split('.').reduce((result: any, part) => result?.[part], en);
  return typeof value === 'string' ? value.replace(/\{(\w+)\}/g, (match, name) => params[name] === undefined ? match : String(params[name])) : key;
};
const compare = (key: string, expected: unknown, actual: unknown) => regressionComparison({ key, expected, actual } as AssertionResult, regressionResultPresentations[key], t);
describe('plugin-declared regression result comparisons', () => {
  it('shows business values for passing scalar checks', () => {
    const rows = compare('template.created', en.regression.fixtures.content, en.regression.fixtures.content)!;
    expect(rows).toHaveLength(1); expect(rows[0].expected).toBe(en.regression.fixtures.content); expect(rows[0].actual).toBe(en.regression.fixtures.content); expect(rows[0].matches).toBe(true);
  });
  it('preserves an observed count difference even when the claimed status is passed', () => {
    const rows = compare('template.cloneCount', { count: 4, failed: 0, independent_uuids: true }, { count: 5, failed: 0, independent_uuids: true })!;
    expect(rows.map(row => row.matches)).toEqual([false, true, true]); expect(rows[0].label).toBe(en.regression.resultFields.count); expect(rows[0].actual).toBe('5');
  });
  it('identifies missing objects without substituting expected data', () => {
    const rows = compare('template.cloneIndependent', [{ name: en.regression.fixtures.name, content: en.regression.fixtures.content }], [])!;
    expect(rows).toHaveLength(2); expect(rows.every(row => !row.recorded && !row.matches)).toBe(true); expect(rows[0].actual).toBe(en.regression.states.unrecorded);
  });
  it('renders nested validation issues using declared locale labels', () => {
    const issue = { field: 'content', rule: 'content_min_length', severity: 'warning' };
    const rows = compare('template.validationWarning', { valid: false, violations: [issue] }, { valid: false, violations: [{ ...issue, severity: 'error' }] })!;
    expect(rows[1].expected).toContain(en.regression.resultEnums.warning); expect(rows[1].actual).toContain(en.regression.resultEnums.error); expect(rows[1].matches).toBe(false);
  });
  it('compares reference identities without exposing UUIDs in business text', () => {
    const first = '11111111-1111-4111-8111-111111111111', second = '22222222-2222-4222-8222-222222222222';
    const rows = compare('metadata.bound', [first], [second])!;
    expect(rows[0].matches).toBe(false); expect(rows[0].actual).toBe(en.regression.resultValues.otherReference); expect(JSON.stringify(rows)).not.toContain(first); expect(JSON.stringify(rows)).not.toContain(second);
  });
  it('reports missing mappings and unknown enum values explicitly', () => {
    expect(compare('unconfigured.assertion', {}, {})).toBeNull();
    const rows = compare('template.rejectedCreate', [{ status: 400, success: false, code: 'VALIDATION_FAILED', field: 'name' }], [{ status: 400, success: false, code: 'unconfigured.code', field: 'name' }])!;
    expect(rows.some(row => !row.recorded && row.actual === en.regression.states.unrecorded)).toBe(true);
  });
});
