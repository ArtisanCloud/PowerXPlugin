import type { RegressionResultField, RegressionResultPresentation } from '@artisan-cloud/plugin-framework-admin/layer/app/types/regression';

const field = (key: string, format: RegressionResultField['format'] = 'text'): RegressionResultField => ({ key, labelKey: 'regression.resultFields.' + key, format });
const enums = (key: string, values: string[]): RegressionResultField => ({ ...field(key, 'enum'), values: Object.fromEntries(values.map(value => [value, 'regression.resultEnums.' + value])) });
const scalar = (label: string, format: RegressionResultField['format'] = 'text'): RegressionResultPresentation => ({ shape: 'scalar', fields: [field(label, format)] });
const record = (...fields: RegressionResultField[]): RegressionResultPresentation => ({ shape: 'record', fields });
const list = (group: string, ...fields: RegressionResultField[]): RegressionResultPresentation => ({ shape: 'list', itemLabelKey: 'regression.resultGroups.' + group, fields });
const violation: RegressionResultField = { ...field('violations', 'items'), fields: [enums('field', ['content', 'name', 'description']), enums('rule', ['content_min_length']), enums('severity', ['warning', 'error'])] };
const rejected = () => list('request', field('status', 'number'), field('success', 'boolean'), enums('field', ['name', 'content']), enums('code', ['VALIDATION_FAILED']));
const search = () => record(field('total', 'number'), field('matching_templates', 'boolean'));

/** Display semantics belong to this plugin. No Template fields enter Framework. */
export const regressionResultPresentations: Record<string, RegressionResultPresentation> = {
  'template.created': scalar('content'), 'template.updated': scalar('content'), 'template.readback': scalar('content'),
  'template.searchName': search(), 'template.searchDescription': search(), 'template.searchCase': search(), 'template.searchAbsent': search(),
  'template.pagination': record(field('counts', 'numbers'), field('totals', 'numbers'), field('complete_without_duplicates', 'boolean')),
  'template.pageOverflow': record(field('count', 'number'), field('total', 'number')),
  'template.deletedRead': scalar('not_found', 'boolean'), 'template.deletedList': record(field('count', 'number'), field('only_control_template', 'boolean')),
  'template.survivor': scalar('content'),
  'template.cloneCount': record(field('count', 'number'), field('failed', 'number'), field('independent_uuids', 'boolean')),
  'template.cloneContents': list('copy', field('name'), field('description'), field('content'), enums('status', ['draft', 'published', 'archived']), enums('review_status', ['pending', 'approved', 'rejected'])),
  'template.cloneIndependent': list('object', field('name'), field('content')),
  'template.partialClone': record(field('created', 'number'), field('failed', 'number'), field('only_missing_source_failed', 'boolean')),
  'template.partialReadback': record(field('count', 'number'), field('source_content'), field('copy_content')),
  'template.rejectedCreate': rejected(), 'template.rejectedUpdate': rejected(),
  'template.noPollution': record(field('count', 'number'), field('only_original', 'boolean'), field('name'), field('description'), field('content')),
  'template.validationWarning': record(field('valid', 'boolean'), violation), 'template.validationStrict': record(field('valid', 'boolean'), violation),
  'template.validationRepaired': record(field('valid', 'boolean'), field('violations', 'number'), field('content')),
  'metadata.bound': { shape: 'references', itemLabelKey: 'regression.resultGroups.tag', fields: [field('tag', 'reference')] },
  'metadata.idempotency': record(field('count', 'number'), field('tag_uuids', 'references')),
  'metadata.readback': { shape: 'references', itemLabelKey: 'regression.resultGroups.tag', fields: [field('tag', 'reference')] },
};
