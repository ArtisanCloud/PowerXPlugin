package knowledge

import "context"

func DefaultKnowledgeCatalog(source string) *KnowledgeCatalog {
	return &KnowledgeCatalog{
		Version: "1",
		Source:  firstNonEmpty(source, "framework_default"),
		Scenes: []KnowledgeScene{
			{Key: "product_specs", Label: "产品规格 / 参数查询", Description: "适合型号、规格、价格、口径等事实精确型问答。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "product_compat", Label: "产品兼容 / 配件关系", Description: "适合兼容矩阵、配件关系、依赖约束等关系型知识。", DefaultStrategyPackage: "K_kg", DefaultBundle: "p3_kg_strong"},
			{Key: "product_selection", Label: "产品选型 / 对比推荐", Description: "适合方案对比、选型建议、推荐理由，需要归纳并保留引用。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "sop", Label: "SOP / 制度 / 产品说明", Description: "适合章节结构清晰的手册、制度、操作说明与产品说明。", DefaultStrategyPackage: "J_hier", DefaultBundle: "p1_general"},
			{Key: "contract_quote", Label: "合同 / 报价 / 条款", Description: "适合合同、报价、政策条款等高风险内容，优先保证证据和纠错。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "research_longdoc", Label: "论文 / 研究 / 长报告", Description: "适合长 PDF、研究报告、方案文档，强调语义切块和章节定位。", DefaultStrategyPackage: "B_semantic_chunking", DefaultBundle: "p1_general"},
			{Key: "ledger_table", Label: "台账 / 清单 / 表格", Description: "适合 Excel/CSV 行级记录、字段过滤和精确查询。", DefaultStrategyPackage: "E_query_transform", DefaultBundle: "p2_high_accuracy"},
			{Key: "support_faq", Label: "客服 FAQ / 使用说明", Description: "适合问答粒度明确、同义问法多的客服知识。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "support_policy", Label: "售后政策 / 规则", Description: "适合退款、保修、售后条款等需要引用证据的政策问答。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "support_troubleshooting", Label: "故障现象 / 排查步骤", Description: "适合现象、原因、处理步骤类知识，需要上下文和步骤链路。", DefaultStrategyPackage: "J_hier", DefaultBundle: "p1_general"},
			{Key: "eng_runbook", Label: "工程 Runbook / 标准操作", Description: "适合运维手册、标准操作、应急流程等步骤化文档。", DefaultStrategyPackage: "J_hier", DefaultBundle: "p1_general"},
			{Key: "api_reference", Label: "API / 接口文档", Description: "适合参数、返回值、字段含义、示例等精确检索。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "data_dictionary", Label: "数据字典 / 字段口径", Description: "适合表结构、字段含义、口径说明与结构化过滤。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "sales_enablement", Label: "销售材料 / 话术 / 竞品", Description: "适合销售话术、竞品对比、案例材料，偏解释归纳。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "compliance_regulation", Label: "法规 / 监管 / 合规政策", Description: "适合法规条款、监管口径、时效性强的合规知识。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "billing_pricing", Label: "计费 / 价格规则", Description: "适合价格、计费、版本差异、例外规则等高准确问答。", DefaultStrategyPackage: "O_crag", DefaultBundle: "p2_high_accuracy"},
			{Key: "meeting_minutes", Label: "会议纪要 / 决议行动项", Description: "适合会议纪要、决议、行动项，强调总结和引用定位。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "project_docs", Label: "项目方案 / 交付文档", Description: "适合需求、设计、方案与交付资料，章节上下文重要。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "ticket_conversations", Label: "工单 / 聊天记录", Description: "适合多轮对话、问题追踪和按话题聚合。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
			{Key: "onboarding_training", Label: "培训 / 入职资料", Description: "适合培训课件、学习路径、入职手册。", DefaultStrategyPackage: "J_hier", DefaultBundle: "p1_general"},
			{Key: "sql_kg", Label: "SQL / 配置 / 依赖关系", Description: "适合系统依赖、配置链路、表关系等图谱强约束知识。", DefaultStrategyPackage: "K_kg", DefaultBundle: "p3_kg_strong"},
			{Key: "custom_expert", Label: "自定义专家模式", Description: "适合专家按需组合策略，仍会保留依赖校验与成本护栏。", DefaultStrategyPackage: "H_fusion", DefaultBundle: "p1_general"},
		},
		StrategyPackages: []StrategyPackage{
			strategyPackage("A_simple", "A Simple RAG（最小闭环）", "基础向量检索", "简单切块 + 向量召回，适合快速验证。", "最快跑通知识库闭环，适合结构清晰、风险较低的 FAQ、SOP 或培训资料。", "字段精确查询、合同报价、强关系依赖。", "p0_basic", []string{"index.dense"}),
			strategyPackage("A0_acl", "A0 Metadata/ACL 优先过滤", "权限与元数据优先", "先做租户/权限/元数据过滤，再做召回，保证合规降噪。", "适合多部门、多角色共用知识库，先按权限、部门、标签过滤，再进入召回。", "单租户低风险验证场景。", "p1_general", nil),
			strategyPackage("C_context_enriched", "C Context Enriched（上下文增强）", "上下文增强", "召回后扩展同章节邻居/父摘要。", "适合条款、制度、SOP 等需要结合前后文才能回答完整的问题。", "只需要单条字段命中的表格或参数库。", "p1_general", []string{"index.hier"}),
			strategyPackage("F_rerank", "F Reranker（重排序）", "重排序精排", "降低相似候选误命中。", "适合召回候选多、相似内容多的知识库，用重排模型提升最终命中质量。", "低成本快速验证或候选很少的库。", "p1_general", []string{"index.dense"}),
			strategyPackage("H_fusion", "H Fusion（融合检索）", "融合检索", "dense+sparse(+kg/hier) 多路召回融合。", "企业知识库通用推荐，同时利用语义召回和关键词召回，兼顾命中率与可解释引用。", "只想最低成本验证，或必须强关系约束的图谱场景。", "p1_general", []string{"index.dense", "index.sparse"}),
			strategyPackage("A1_routing", "A1 Query Routing（查询路由）", "查询路由", "按意图/领域把 query 路由到不同索引通道或空间。", "适合多个知识域混用，把问题先分流到合同、产品、SOP、接口等不同空间或索引。", "单一知识空间、单一文档类型。", "p1_general", nil),
			strategyPackage("E_query_transform", "E Query Transformation（查询转换）", "查询理解与改写", "同义扩展/结构化抽取/纠错。", "适合用户问法口语化、同义词多、需要抽取型号/时间/金额/字段条件的场景。", "标准 FAQ 或标题关键词已经很稳定的库。", "p1_general", []string{"index.structured_fields"}),
			strategyPackage("G_rse", "G RSE（语义扩展重排）", "术语扩展重排", "语义扩展 + 重排，适合术语多的场景。", "适合专业术语、型号、缩写、领域词很多的产品、接口、数据字典知识。", "普通 SOP 或低成本验证。", "p2_high_accuracy", []string{"index.dense"}),
			strategyPackage("I_hyde", "I HyDE（假设文档检索）", "假设答案检索", "生成假设答案后再向量检索。", "适合问题很抽象、缺少关键词的研究资料或长报告，用假设答案扩大召回。", "高风险事实精确问答。", "p1_general", []string{"index.dense"}),
			strategyPackage("L_feedback", "L Feedback Loop（反馈闭环）", "反馈闭环治理", "标注→再加工→回归评估。", "适合上线后的质量治理，把用户反馈转成重分块、重索引或策略回滚依据。", "一次性临时调试空间。", "p1_general", nil),
			strategyPackage("M_adaptive", "M Adaptive RAG（自适应策略）", "自适应检索策略", "按置信度/成本动态启用策略。", "适合流量大、问题类型多、需要按成本和置信度动态选择召回/重排/纠错的场景。", "简单固定流程知识库。", "p2_high_accuracy", nil),
			strategyPackage("N_self_rag", "N Self RAG（自反思回路）", "证据自检回路", "自检证据与冲突，触发二次检索。", "适合合同、合规、价格等需要检查证据是否充分、是否冲突的高风险问答。", "低成本、大吞吐普通 FAQ。", "p2_high_accuracy", nil),
			strategyPackage("B_semantic_chunking", "B Semantic Chunking（语义切块）", "语义切块", "按语义边界切块，适合论文/长报告。", "适合长论文、研究报告、项目方案，按语义边界切分并保留章节锚点。", "短 FAQ、表格台账、字段精确查询。", "p1_general", []string{"index.dense"}),
			strategyPackage("J_hier", "J Hierarchical Indices（层次索引）", "层次索引", "先章节/摘要，再下钻 chunk。", "适合 SOP、Runbook、长报告等章节结构清晰的文档，先定位章节摘要，再下钻到具体片段。", "字段精确查询、表格台账、强关系依赖。", "p1_general", []string{"index.hier"}),
			strategyPackage("D_doc_augmentation", "D Doc Augmentation（离线增强）", "离线文档增强", "生成摘要/关键词/实体标签等增强字段。", "适合合同、报价、产品说明等字段密集文档，提前生成摘要、关键词、实体标签。", "临时调试或无需离线加工的简单资料。", "p2_high_accuracy", []string{"index.structured_fields"}),
			strategyPackage("A2_time_aware", "A2 Time-aware（时间/版本）", "时间与版本感知", "对版本/生效时间做过滤或权重衰减。", "适合政策、报价、产品版本频繁变化的知识，按生效时间和版本线控制答案。", "静态资料或无版本字段的文档。", "p2_high_accuracy", []string{"index.time_fields"}),
			strategyPackage("K_kg", "K Knowledge Graph（知识图谱）", "知识图谱约束", "实体/关系召回与约束。", "适合兼容关系、系统依赖、SQL/配置链路等关系驱动知识，用实体关系约束召回和解释。", "普通说明文、FAQ 或没有实体关系抽取的资料。", "p3_kg_strong", []string{"index.kg"}),
			strategyPackage("O_crag", "O CRAG（纠错）", "证据纠错检索", "证据不足/冲突时触发纠错检索。", "适合合同、报价、政策、产品参数等高风险问答，证据不足或冲突时触发纠错检索。", "低成本泛问答或只做快速验证。", "p2_high_accuracy", []string{"index.dense", "index.sparse"}),
		},
		StrategyBundles: []StrategyBundle{
			{Key: "p0_basic", Label: "P0 基础（最小闭环）", Description: "最少干预：仅 dense 召回 + 引用返回。"},
			{Key: "p1_general", Label: "P1 通用推荐（企业默认）", Description: "hybrid + RRF + 轻量 rerank + contextual。"},
			{Key: "p2_high_accuracy", Label: "P2 高准确/合规（证据优先）", Description: "sparse 优先 + hybrid + CRAG/可选 self-check。"},
			{Key: "p3_kg_strong", Label: "P3 KG 约束（关系驱动）", Description: "KG recall/filter + hybrid + cite。"},
		},
		Metadata: map[string]any{
			"authority": "PowerX scene_strategy_catalog compatible fallback",
		},
	}
}

func strategyPackage(key, label, display, summary, useCase, notFor, profile string, indexDeps []string) StrategyPackage {
	return StrategyPackage{
		Key:                   key,
		Label:                 label,
		DisplayLabel:          display,
		Summary:               summary,
		UseCase:               useCase,
		NotFor:                notFor,
		RecommendedProfileKey: profile,
		Dependencies:          StrategyDependency{Index: indexDeps},
	}
}

func ProviderCatalog(ctx context.Context, provider KnowledgeProvider) (*KnowledgeCatalog, error) {
	if provider == nil {
		return DefaultKnowledgeCatalog("framework_default"), nil
	}
	return provider.Catalog(ctx)
}
