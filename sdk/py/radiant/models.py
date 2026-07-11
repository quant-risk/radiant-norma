"""
Pydantic models mirroring OpenAPI v3.36.2 spec.

Auto-generated from docs/openapi/v1.yaml.
"""
from __future__ import annotations

from datetime import datetime
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class ValidationErrorItem(BaseModel):
    code: str
    severity: str
    field: Optional[str] = None
    message: str
    position: Optional[str] = None


class ComplexityScore(BaseModel):
    score: float
    num_operacoes: int = 0
    num_participantes: int = 0
    estimated_api_calls: int = 0
    estimated_time_ms: int = 0


class HealthResponse(BaseModel):
    status: str
    version: str


class ReadyResponse(BaseModel):
    status: str
    db: str


class SchemasResponse(BaseModel):
    cadocs: List[str]
    total: int


class SchemaVersion(BaseModel):
    cadoc: str
    version: str
    effective_from: Optional[str] = None
    source_uri: Optional[str] = None


class VersionsResponse(BaseModel):
    cadoc: str
    versions: List[str]
    total: int


class SchemaInfo(BaseModel):
    cadoc_code: str
    latest_version: Optional[str] = None
    effective_from: Optional[str] = None
    source_uri: Optional[str] = None
    supported_versions: List[str] = []
    field_count: int = 0
    complexity: Optional[ComplexityScore] = None


class SchemaListResponse(BaseModel):
    schemas: List[SchemaInfo]
    total: int


class Rule(BaseModel):
    code: str
    description: str
    severity: str
    category: Optional[str] = None
    required_docs: List[str] = []


class RuleListResponse(BaseModel):
    cadoc: str
    rules: List[Rule]
    total: int


class RulesListResponse(BaseModel):
    cadocs: List[str]
    total: int


class Participante(BaseModel):
    id: str
    tipo: Optional[str] = None
    nome: Optional[str] = None
    cpf: Optional[str] = None
    cnpj: Optional[str] = None
    rating: Optional[str] = None


class Operacao(BaseModel):
    id: str
    tipo: Optional[str] = None
    valor: Optional[float] = None
    prazo: Optional[int] = None
    taxa: Optional[float] = None


class GenerateRequest(BaseModel):
    cadoc_code: str
    if_id: str
    cnpj: Optional[str] = None
    nome_if: Optional[str] = None
    versao_layout: Optional[str] = None
    data_base: Optional[str] = None
    extra: Optional[Dict[str, Any]] = None
    participantes: List[Participante] = []
    operacoes: List[Operacao] = []
    source: Optional[str] = None  # manual | api | file | db | mcp


class GeneratedDoc(BaseModel):
    xml: Optional[str] = None
    xml_hash: Optional[str] = None
    explain: Optional[Dict[str, Any]] = None


class GenerateResponse(BaseModel):
    cadoc_code: str
    data_base: Optional[str] = None
    generated: Optional[GeneratedDoc] = None
    status: str  # generated | error
    message: Optional[str] = None


class Field(BaseModel):
    name: str
    type: str
    required: bool = False
    description: Optional[str] = None
    example: Optional[Any] = None


class FieldsResponse(BaseModel):
    cadoc_code: str
    fields: List[Field] = []
    versions: List[str] = []
    complexity: Optional[ComplexityScore] = None


class AdapterInfo(BaseModel):
    type: str  # manual | api | file | db | mcp
    name: str


class AdaptersResponse(BaseModel):
    adapters: List[AdapterInfo]


class SourceConfigResponse(BaseModel):
    cadoc_code: str
    source_type: str
    source_name: Optional[str] = None
    fields: List[Field] = []
    status: str
    message: Optional[str] = None


class BatchResult(BaseModel):
    cadoc_code: str
    status: str
    xml_hash: Optional[str] = None
    message: Optional[str] = None


class CrossDocError(BaseModel):
    code: str
    severity: str  # error | warning
    message: str
    involved_docs: List[str] = []


class BatchGenerateRequest(BaseModel):
    cadocs: List[GenerateRequest]
    run_crossdoc: bool = False


class BatchGenerateResponse(BaseModel):
    results: List[BatchResult]
    crossdoc_errors: List[CrossDocError] = []
    crossdoc_warnings: List[CrossDocError] = []
    passed: bool
    message: Optional[str] = None


class GenerationHistoryItem(BaseModel):
    id: str
    cadoc_code: str
    data_base: str
    generated_at: datetime
    sha256: Optional[str] = None
    status: str
    passed: bool


class GenerationHistoryResponse(BaseModel):
    items: List[GenerationHistoryItem]
    page: int
    per_page: int
    total: int


class CrossDocInput(BaseModel):
    cadoc_code: str
    xml: str
    data_base: Optional[str] = None


class CrossDocValidateRequest(BaseModel):
    documents: List[CrossDocInput]


class CrossDocValidateResponse(BaseModel):
    passed: bool
    errors: List[CrossDocError] = []
    warnings: List[CrossDocError] = []
    rules_executed: int = 0
    duration_ms: int = 0


class CrossDocRule(BaseModel):
    code: str
    description: str
    severity: str
    required_docs: List[str] = []


class CrossDocRulesResponse(BaseModel):
    rules: List[CrossDocRule]
    total: int


class ValidateRequest(BaseModel):
    cadoc_code: str
    data_base: Optional[str] = None
    xml: str
    content_type: str = "application/xml"


class ValidateResponse(BaseModel):
    cadoc_code: str
    data_base: Optional[str] = None
    xml_hash: Optional[str] = None
    passed: bool
    errors: List[ValidationErrorItem] = []
    warnings: List[ValidationErrorItem] = []
    executed_at: Optional[datetime] = None
    duration_ms: int = 0
    disabled_rules: List[str] = []


class StaSubmitRequest(BaseModel):
    cadoc_code: str
    data_base: Optional[str] = None
    xml: str
    xml_hash: Optional[str] = None
    protocolo_sta: Optional[str] = None


class StaSubmitResponse(BaseModel):
    protocolo: str
    status: str  # submitted | accepted | pending
    submitted_at: Optional[datetime] = None


class StaDisponiveisDocumento(BaseModel):
    protocolo: str
    cadoc_code: str
    data_base: str
    received_at: Optional[datetime] = None
    size_bytes: int = 0


class StaDisponiveisResponse(BaseModel):
    documentos: List[StaDisponiveisDocumento]
    total: int


class StaSituacaoResponse(BaseModel):
    protocolo: str
    status: str  # pending | accepted | rejected | error
    details: Optional[str] = None
    updated_at: Optional[datetime] = None


class RadarAlert(BaseModel):
    id: str
    cadoc: str
    version_before: Optional[str] = None
    version_after: Optional[str] = None
    change_type: Optional[str] = None
    description: Optional[str] = None
    source: Optional[str] = None
    status: str  # open | resolved | dismissed
    created_at: Optional[datetime] = None
    resolved_at: Optional[datetime] = None


class RadarAlertsResponse(BaseModel):
    alerts: List[RadarAlert]
    total: int
    page: int = 1
    per_page: int = 20


class L4SubmissionSnapshot(BaseModel):
    id: Optional[str] = None
    cadoc_code: Optional[str] = None
    data_base: Optional[str] = None
    xml_hash: Optional[str] = None
    submitted_at: Optional[datetime] = None
    passed: Optional[bool] = None


class L4FailedRule(BaseModel):
    code: str
    severity: Optional[str] = None
    message: Optional[str] = None


class L4FieldChange(BaseModel):
    cadoc_code: str
    field: str
    previous: float = 0.0
    current: float = 0.0
    delta_pct: float = 0.0


class L4Alert(BaseModel):
    type: str
    code: Optional[str] = None
    severity: Optional[str] = None
    message: Optional[str] = None


class L4Comparison(BaseModel):
    current: Optional[L4SubmissionSnapshot] = None
    previous: Optional[L4SubmissionSnapshot] = None
    new_failures: List[L4FailedRule] = []
    fixed_rules: List[L4FailedRule] = []
    changed_fields: List[L4FieldChange] = []
    alerts: List[L4Alert] = []


class Envio(BaseModel):
    id: str
    if_id: str
    cadoc_code: str
    data_base: str
    xml_hash: Optional[str] = None
    status: str
    created_at: Optional[datetime] = None
    submitted_at: Optional[datetime] = None
    accepted_at: Optional[datetime] = None


class EnviosResponse(BaseModel):
    envios: List[Envio]
    total: int
    page: int = 1
    per_page: int = 20


class EnviosStats(BaseModel):
    total: int = 0
    by_status: Dict[str, int] = {}
    by_cadoc: Dict[str, int] = {}
    acceptance_rate: float = 0.0


class AuditLogEntry(BaseModel):
    seq: int
    timestamp: datetime
    if_id: str
    action: str
    resource: Optional[str] = None
    prev_hash: Optional[str] = None
    hash: str
    metadata: Optional[Dict[str, Any]] = None


class AuditLogResponse(BaseModel):
    entries: List[AuditLogEntry]
    total: int
    page: int = 1
    per_page: int = 50


class KPIResponse(BaseModel):
    period: str = "30d"
    total_validations: int = 0
    error_rate: float = 0.0
    top_cadoc: Optional[str] = None
    avg_duration_ms: float = 0.0
    trend: str = "stable"  # improving | stable | degrading


class HeatmapResponse(BaseModel):
    data: List[List[int]] = []
    day_labels: List[str] = []
    hour_labels: List[int] = []


class TopFailingRule(BaseModel):
    code: str
    cadoc: str
    fail_count: int = 0
    fail_rate: float = 0.0
    avg_field_value: Optional[Any] = None


class TopFailingRulesResponse(BaseModel):
    period: str
    rules: List[TopFailingRule] = []
    total: int


class Recommendation(BaseModel):
    id: str
    type: str
    title: str
    description: str
    cadoc: Optional[str] = None
    rule_codes: List[str] = []
    status: str = "pending"  # pending | acknowledged
    created_at: Optional[datetime] = None


class RecommendationsResponse(BaseModel):
    recommendations: List[Recommendation]
    total: int
