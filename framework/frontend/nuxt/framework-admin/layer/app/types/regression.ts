/** Plugin-owned business explanations and form fields for a registered scenario. */
export interface RegressionCasePresentation {
  expectationKeys: string[];
  impactKey: string;
  fields: Array<{
    key: string;
    labelKey: string;
    control: 'text' | 'textarea';
    helpKey?: string;
  }>;
  results?: Record<string, RegressionResultPresentation>;
}

/** Explicit plugin-owned mapping; the renderer never guesses business fields. */
export interface RegressionResultField {
  key: string;
  labelKey: string;
  format: 'text' | 'number' | 'boolean' | 'enum' | 'numbers' | 'items' | 'reference' | 'references';
  values?: Record<string, string>;
  fields?: RegressionResultField[];
}
export interface RegressionResultPresentation {
  shape: 'scalar' | 'record' | 'list' | 'references';
  fields: RegressionResultField[];
  itemLabelKey?: string;
}
