import type { AssertionResult } from '@artisan-cloud/plugin-framework-client';
import type { RegressionResultField, RegressionResultPresentation } from '../types/regression';

export interface RegressionComparisonRow {
  label: string;
  expected: string;
  actual: string;
  matches: boolean;
  recorded: boolean;
}
type Translate = (key: string, params?: Record<string, string | number>) => string;
const object = (value: unknown): value is Record<string, unknown> => !!value && typeof value === 'object' && !Array.isArray(value);
function equal(a: unknown, b: unknown): boolean {
  if (Array.isArray(a) && Array.isArray(b)) return a.length === b.length && a.every((v, i) => equal(v, b[i]));
  if (object(a) && object(b)) return Object.keys(a).length === Object.keys(b).length && Object.keys(a).every(key => Object.hasOwn(b, key) && equal(a[key], b[key]));
  return a === b;
}
function format(value: unknown, field: RegressionResultField, t: Translate): string | null {
  if (field.format === 'text') return typeof value === 'string' ? value : null;
  if (field.format === 'number') return typeof value === 'number' && Number.isFinite(value) ? String(value) : null;
  if (field.format === 'boolean') return typeof value === 'boolean' ? t(value ? 'regression.resultValues.yes' : 'regression.resultValues.no') : null;
  if (field.format === 'enum') return typeof value === 'string' && field.values?.[value] ? t(field.values[value]) : null;
  if (field.format === 'numbers') return Array.isArray(value) && value.every(item => typeof item === 'number') ? value.join(' / ') : null;
  if (field.format === 'references') return Array.isArray(value) && value.every(item => typeof item === 'string') ? t('regression.resultValues.referencesCount', { count: value.length }) : null;
  if (field.format === 'items' && Array.isArray(value) && field.fields?.length) {
    const entries: string[] = [];
    for (const item of value) {
      if (!object(item)) return null;
      const parts: string[] = [];
      for (const child of field.fields) {
        const rendered = format(item[child.key], child, t);
        if (rendered === null) return null;
        parts.push(t('regression.resultValues.fieldValue', { field: t(child.labelKey), value: rendered }));
      }
      entries.push(parts.join(t('regression.resultValues.separator')));
    }
    return entries.length ? entries.join('\n') : t('regression.resultValues.none');
  }
  return null;
}
export function regressionComparison(assertion: AssertionResult, presentation: RegressionResultPresentation | undefined, t: Translate): RegressionComparisonRow[] | null {
  if (!presentation || !presentation.fields.length) return null;
  const rows: RegressionComparisonRow[] = [];
  const add = (field: RegressionResultField, expected: unknown, actual: unknown, prefix = '') => {
    const isReference = field.format === 'reference';
    const knownExpected = isReference ? typeof expected === 'string' : format(expected, field, t) !== null;
    const knownActual = isReference ? typeof actual === 'string' : format(actual, field, t) !== null;
    rows.push({
      label: prefix ? t('regression.resultValues.groupField', { group: prefix, field: t(field.labelKey) }) : t(field.labelKey),
      expected: isReference && knownExpected ? t('regression.resultValues.selectedReference') : format(expected, field, t) ?? t('regression.states.unrecorded'),
      actual: isReference && knownActual ? t(expected === actual ? 'regression.resultValues.selectedReference' : 'regression.resultValues.otherReference') : format(actual, field, t) ?? t('regression.states.unrecorded'),
      matches: knownExpected && knownActual && equal(expected, actual), recorded: knownExpected && knownActual,
    });
  };
  if (presentation.shape === 'scalar') add(presentation.fields[0], assertion.expected, assertion.actual);
  else if (presentation.shape === 'record') {
    if (!object(assertion.expected) || !object(assertion.actual)) return null;
    for (const field of presentation.fields) add(field, assertion.expected[field.key], assertion.actual[field.key]);
  } else {
    if (!Array.isArray(assertion.expected) || !Array.isArray(assertion.actual) || !presentation.itemLabelKey) return null;
    const size = Math.max(assertion.expected.length, assertion.actual.length);
    for (let index = 0; index < size; index++) {
      const prefix = t(presentation.itemLabelKey, { index: index + 1 });
      if (presentation.shape === 'references') add(presentation.fields[0], assertion.expected[index], assertion.actual[index], prefix);
      else {
        const expected = assertion.expected[index], actual = assertion.actual[index];
        if (expected !== undefined && !object(expected) || actual !== undefined && !object(actual)) return null;
        for (const field of presentation.fields) add(field, expected?.[field.key], actual?.[field.key], prefix);
      }
    }
  }
  return rows;
}
