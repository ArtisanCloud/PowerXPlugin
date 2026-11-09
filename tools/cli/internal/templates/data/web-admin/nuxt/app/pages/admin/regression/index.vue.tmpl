<script setup lang="ts">
import {computed, ref, watch} from 'vue';
import {useAuth} from '~/composables/useAuth';
import {useI18n} from '#imports';
import type {CaseDefinition} from '@artisan-cloud/plugin-framework-client';
import PXRegressionWorkbench from '@artisan-cloud/plugin-framework-admin/layer/app/components/powerx/PXRegressionWorkbench.vue';
import type {RegressionCasePresentation} from '@artisan-cloud/plugin-framework-admin/layer/app/types/regression';
import {useRegressionApi} from '~/composables/api/useRegression';
import {useUserStore} from '~/stores/user';
import {regressionResultPresentations} from '~/composables/regressionPresentation';
const user=useUserStore(),client=useRegressionApi(),{t}=useI18n();
const can=(action:string)=>computed(()=>user.isRoot||(user.context?.permissions||[]).some((code:string)=>code==='*'||code==='regression.runs:'+action));
const read=can('read'),invoke=can('invoke'),manage=can('manage');
const auth=useAuth(),sessionRevision=ref(0);
watch(()=>auth.token.value,()=>sessionRevision.value++);
const identityKey=computed(()=>`${user.context?.current_tenant_uuid||''}:${user.userUuid||''}:${sessionRevision.value}`);
function inputs(definition:CaseDefinition){
 if(!definition.key.startsWith('template.')||!presentations[definition.key])return {};
 const values:Record<string,string>={name:t('regression.fixtures.name'),description:t('regression.fixtures.description'),content:t(definition.key==='template.validation'?'regression.fixtures.validationContent':'regression.fixtures.content'),updated_content:t('regression.fixtures.updatedContent')};
 return Object.fromEntries(presentations[definition.key].fields.map(field=>[field.key,values[field.key]]));
}
const templateFields:RegressionCasePresentation['fields']=[
 {key:'name',labelKey:'regression.inputs.templateName',control:'text'},
 {key:'description',labelKey:'regression.inputs.templateDescription',control:'text'},
 {key:'content',labelKey:'regression.inputs.initialContent',control:'textarea'},
];
function templatePresentation(key:string,expectations:string[],updated=false):RegressionCasePresentation{
 return {expectationKeys:expectations.map(value=>'regression.assertions.'+value),impactKey:'regression.scenarios.'+key+'.impact',fields:[...templateFields,...(updated?[{key:'updated_content',labelKey:'regression.inputs.updatedContent',control:'textarea' as const}]:[])]};
}
const presentations:Record<string,RegressionCasePresentation>={
 'template.lifecycle':{
  expectationKeys:['regression.assertions.created','regression.assertions.updated','regression.assertions.readback'],
  impactKey:'regression.scenarios.template.impact',
  fields:[
   {key:'name',labelKey:'regression.inputs.templateName',control:'text'},
   {key:'description',labelKey:'regression.inputs.templateDescription',control:'text'},
   {key:'content',labelKey:'regression.inputs.initialContent',control:'textarea'},
   {key:'updated_content',labelKey:'regression.inputs.updatedContent',control:'textarea'},
  ],
 },
 'template.search':templatePresentation('search',['searchName','searchDescription','searchCase','searchAbsent']),
 'template.pagination':templatePresentation('pagination',['pagination','pageOverflow']),
 'template.deletion':templatePresentation('deletion',['deletedRead','deletedList','survivor']),
 'template.clone':templatePresentation('clone',['cloneCount','cloneContents','cloneIndependent'],true),
 'template.partialClone':templatePresentation('partialClone',['partialClone','partialReadback']),
 'template.invalidInput':templatePresentation('invalidInput',['rejectedCreate','rejectedUpdate','noPollution']),
 'template.validation':{
  ...templatePresentation('validation',['validationWarning','validationStrict','validationRepaired']),
  fields:templateFields.map(field=>field.key==='content'?{...field,labelKey:'regression.inputs.repairedContent'}:field),
 },
 'metadata.tag_bindings':{
  expectationKeys:['regression.assertions.binding','regression.assertions.idempotency'],
  impactKey:'regression.scenarios.metadata.impact',
  fields:[
   {key:'resource_type',labelKey:'regression.inputs.resourceType',control:'text'},
   {key:'tag_uuid',labelKey:'regression.inputs.testTag',helpKey:'regression.inputs.testTagHelp',control:'text'},
  ],
 },
};
const describeCase=(definition:CaseDefinition)=>{
 const presentation=presentations[definition.key];
 return presentation?{...presentation,results:Object.fromEntries(definition.required_assertion_keys.filter(key=>regressionResultPresentations[key]).map(key=>[key,regressionResultPresentations[key]]))}:undefined;
};
</script>
<template><PXRegressionWorkbench :client="client" :can-read="read" :can-invoke="invoke" :can-manage="manage" :identity-key="identityKey" :default-inputs="inputs" :describe-case="describeCase"/></template>
