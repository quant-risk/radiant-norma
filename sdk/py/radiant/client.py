"""
Radiant Norma API Client.

Usage::

    from radiant import Client

    c = Client(
        base_url="https://api.radiantrisk.com/v1",
        auth_token="your-jwt-token",
    )

    health = c.healthz()
    schemas = c.list_schemas_v2()
    resp = c.generate_cadoc("3040", {
        "if_id": "if_demo",
        "cnpj": "12.345.678/0001-90",
    })
"""
from __future__ import annotations

from typing import Any, Dict, List, Optional, Union
from urllib.parse import quote

import requests
from requests import Response

from radiant.exceptions import HTTPError
from radiant.models import (
    AdapterInfo,
    AdaptersResponse,
    AuditLogResponse,
    BatchGenerateRequest,
    BatchGenerateResponse,
    CrossDocInput,
    CrossDocRule,
    CrossDocRulesResponse,
    CrossDocValidateRequest,
    CrossDocValidateResponse,
    FieldsResponse,
    GenerateRequest,
    GenerateResponse,
    GenerationHistoryItem,
    GenerationHistoryResponse,
    HealthResponse,
    L4Comparison,
    RadarAlert,
    RadarAlertsResponse,
    ReadyResponse,
    RuleListResponse,
    RulesListResponse,
    SchemaInfo,
    SchemaListResponse,
    SchemaVersion,
    SchemasResponse,
    ValidateResponse,
    VersionsResponse,
)


class Client:
    """
    Radiant Norma API client.

    Args:
        base_url: API base URL. Defaults to "https://api.radiantrisk.com/v1".
        auth_token: JWT Bearer token.
        if_id: Optional X-IF-ID header (fallback auth).
        timeout: Request timeout in seconds. Defaults to 30.
    """

    def __init__(
        self,
        base_url: str = "https://api.radiantrisk.com/v1",
        auth_token: str = "",
        if_id: str = "",
        timeout: int = 30,
    ):
        self.base_url = base_url.rstrip("/")
        self.auth_token = auth_token
        self.if_id = if_id
        self.timeout = timeout
        self._session = requests.Session()
        self._session.headers.update({"Accept": "application/json"})

    def _request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
    ) -> Response:
        """Execute HTTP request with auth headers and error handling."""
        url = f"{self.base_url}/{path.lstrip('/')}"
        headers = {}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        if self.if_id:
            headers["X-IF-ID"] = self.if_id

        resp = self._session.request(
            method=method,
            url=url,
            params=params,
            json=json,
            headers=headers,
            timeout=self.timeout,
        )
        if resp.status_code >= 400:
            raise HTTPError(
                status_code=resp.status_code,
                code=_get_error_code(resp),
                message=_get_error_message(resp),
            )
        return resp

    # -------------------------------------------------------------------------
    # Health
    # -------------------------------------------------------------------------

    def healthz(self) -> HealthResponse:
        """GET /healthz — liveness probe."""
        resp = self._request("GET", "/healthz")
        return HealthResponse.model_validate(resp.json())

    def readyz(self) -> ReadyResponse:
        """GET /readyz — readiness probe (checks DB)."""
        resp = self._request("GET", "/readyz")
        return ReadyResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Schema
    # -------------------------------------------------------------------------

    def list_schemas(self) -> SchemasResponse:
        """GET /schemas — basic CADOC code list."""
        resp = self._request("GET", "/schemas")
        return SchemasResponse.model_validate(resp.json())

    def list_schemas_v2(self) -> SchemaListResponse:
        """GET /schema — enriched schema list with complexity and versions."""
        resp = self._request("GET", "/schema")
        return SchemaListResponse.model_validate(resp.json())

    def get_schema(self, cadoc: str) -> SchemaVersion:
        """GET /schemas/{cadoc} — effective schema for a CADOC."""
        resp = self._request("GET", f"/schemas/{quote(cadoc, safe='')}")
        return SchemaVersion.model_validate(resp.json())

    def list_versions(self, cadoc: str) -> VersionsResponse:
        """GET /schemas/{cadoc}/versions — all layout versions for a CADOC."""
        resp = self._request("GET", f"/schemas/{cadoc}/versions")
        return VersionsResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Rules
    # -------------------------------------------------------------------------

    def list_rules(self) -> RulesListResponse:
        """GET /rules — CADOCs with rules available."""
        resp = self._request("GET", "/rules")
        return RulesListResponse.model_validate(resp.json())

    def list_rules_by_cadoc(self, cadoc: str) -> RuleListResponse:
        """GET /rules/{cadoc} — all rules for a specific CADOC."""
        resp = self._request("GET", f"/rules/{quote(cadoc, safe='')}")
        return RuleListResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Validation
    # -------------------------------------------------------------------------

    def validate(
        self,
        cadoc: str,
        xml: str,
        data_base: Optional[str] = None,
        content_type: str = "application/xml",
    ) -> ValidateResponse:
        """POST /validate — validate a CADOC document (L1 XSD + L2 Semantic + L3 Cross-doc)."""
        payload: Dict[str, Any] = {
            "cadoc_code": cadoc,
            "xml": xml,
            "content_type": content_type,
        }
        if data_base:
            payload["data_base"] = data_base

        resp = self._request("POST", "/validate", json=payload)
        return ValidateResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Generation
    # -------------------------------------------------------------------------

    def generate_cadoc(
        self,
        cadoc: str,
        request: Dict[str, Any],
    ) -> GenerateResponse:
        """
        POST /generate/{cadoc} — generate a CADOC from Canonical Model.

        Args:
            cadoc: CADOC code (e.g. "3040", "4111").
            request: GenerateRequest payload as dict.
                Required: cadoc_code, if_id.
                Optional: cnpj, nome_if, versao_layout, data_base, participantes, operacoes.
        """
        resp = self._request("POST", f"/generate/{quote(cadoc, safe='')}", json=request)
        return GenerateResponse.model_validate(resp.json())

    def list_generate_fields(self, cadoc: str) -> FieldsResponse:
        """GET /generate/{cadoc}/fields — required fields for generating a CADOC."""
        resp = self._request("GET", f"/generate/{quote(cadoc, safe='')}/fields")
        return FieldsResponse.model_validate(resp.json())

    def list_source_adapters(self) -> AdaptersResponse:
        """GET /generate/adapters — available data source adapters."""
        resp = self._request("GET", "/generate/adapters")
        return AdaptersResponse.model_validate(resp.json())

    def generate_batch(self, request: Dict[str, Any]) -> BatchGenerateResponse:
        """POST /generate/batch — generate multiple CADOCs in one call."""
        resp = self._request("POST", "/generate/batch", json=request)
        return BatchGenerateResponse.model_validate(resp.json())

    def list_generate_history(
        self,
        page: int = 1,
        per_page: int = 20,
        cadoc: Optional[str] = None,
    ) -> GenerationHistoryResponse:
        """GET /generate/history — generation history for the authenticated IF."""
        params: Dict[str, Any] = {"page": page, "per_page": per_page}
        if cadoc:
            params["cadoc"] = cadoc
        resp = self._request("GET", "/generate/history", params=params)
        return GenerationHistoryResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Cross-Doc
    # -------------------------------------------------------------------------

    def list_crossdoc_rules(self) -> CrossDocRulesResponse:
        """GET /crossdoc/rules — all available cross-document rules."""
        resp = self._request("GET", "/crossdoc/rules")
        return CrossDocRulesResponse.model_validate(resp.json())

    def crossdoc_validate(
        self,
        documents: List[Dict[str, str]],
    ) -> CrossDocValidateResponse:
        """
        POST /crossdoc/validate — validate consistency between related CADOCs.

        Args:
            documents: List of {"cadoc_code": "3040", "xml": "<xml>..."}.
        """
        payload = {"documents": documents}
        resp = self._request("POST", "/crossdoc/validate", json=payload)
        return CrossDocValidateResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # STA
    # -------------------------------------------------------------------------

    def sta_submit(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """POST /sta/submit — submit a document to BACEN STA."""
        resp = self._request("POST", "/sta/submit", json=request)
        return resp.json()

    def sta_disponiveis(self) -> Dict[str, Any]:
        """GET /sta/disponiveis — list documents available for download from STA."""
        resp = self._request("GET", "/sta/disponiveis")
        return resp.json()

    def sta_situacao(self, protocolo: str) -> Dict[str, Any]:
        """POST /sta/situacao — query STA status for a protocol."""
        resp = self._request("POST", "/sta/situacao", json={"protocolo": protocolo})
        return resp.json()

    # -------------------------------------------------------------------------
    # Radar
    # -------------------------------------------------------------------------

    def list_radar_alerts(
        self,
        status: Optional[str] = None,
        page: int = 1,
        per_page: int = 20,
    ) -> RadarAlertsResponse:
        """GET /radar/alerts — list layout change alerts."""
        params: Dict[str, Any] = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        resp = self._request("GET", "/radar/alerts", params=params)
        return RadarAlertsResponse.model_validate(resp.json())

    def get_radar_alert(self, alert_id: str) -> RadarAlert:
        """GET /radar/alerts/{id} — details of a specific alert."""
        resp = self._request("GET", f"/radar/alerts/{quote(alert_id, safe='')}")
        return RadarAlert.model_validate(resp.json())

    def resolve_radar_alert(self, alert_id: str, resolution: str = "") -> Dict[str, Any]:
        """POST /radar/alerts/{id}/resolve — mark an alert as resolved."""
        payload: Dict[str, Any] = {}
        if resolution:
            payload["resolution"] = resolution
        resp = self._request("POST", f"/radar/alerts/{quote(alert_id, safe='')}/resolve", json=payload)
        return resp.json()

    def trigger_radar_scan(self, cadoc: Optional[str] = None) -> Dict[str, Any]:
        """POST /radar/scan — trigger an on-demand radar scan."""
        payload: Dict[str, Any] = {}
        if cadoc:
            payload["cadoc"] = cadoc
        resp = self._request("POST", "/radar/scan", json=payload)
        return resp.json()

    # -------------------------------------------------------------------------
    # L4 / Audit
    # -------------------------------------------------------------------------

    def l4_compare(self, envio_id: str) -> L4Comparison:
        """GET /l4/compare?envio_id=<uuid> — compare a submission with its previous version."""
        resp = self._request("GET", "/l4/compare", params={"envio_id": envio_id})
        return L4Comparison.model_validate(resp.json())

    def list_envios(
        self,
        status: Optional[str] = None,
        page: int = 1,
        per_page: int = 20,
    ) -> Dict[str, Any]:
        """GET /envios — list submissions for the authenticated IF."""
        params: Dict[str, Any] = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        resp = self._request("GET", "/envios", params=params)
        return resp.json()

    def audit_log(
        self,
        page: int = 1,
        per_page: int = 50,
    ) -> AuditLogResponse:
        """GET /audit_log — tamper-evident audit log entries."""
        resp = self._request("GET", "/audit_log", params={"page": page, "per_page": per_page})
        return AuditLogResponse.model_validate(resp.json())

    # -------------------------------------------------------------------------
    # Insights
    # -------------------------------------------------------------------------

    def insights_kpis(self) -> Dict[str, Any]:
        """GET /insights/kpis — main KPIs."""
        resp = self._request("GET", "/insights/kpis")
        return resp.json()

    def insights_heatmap(self) -> Dict[str, Any]:
        """GET /insights/heatmap — error heatmap by hour/weekday."""
        resp = self._request("GET", "/insights/heatmap")
        return resp.json()

    def insights_top_failing_rules(
        self,
        cadoc: Optional[str] = None,
        period: str = "30d",
    ) -> Dict[str, Any]:
        """GET /insights/rules/top-failing — top 10 failing rules."""
        params: Dict[str, Any] = {"period": period}
        if cadoc:
            params["cadoc"] = cadoc
        resp = self._request("GET", "/insights/rules/top-failing", params=params)
        return resp.json()

    def close(self) -> None:
        """Close the underlying session."""
        self._session.close()

    def __enter__(self) -> "Client":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


# -------------------------------------------------------------------------
# Helpers
# -------------------------------------------------------------------------

def _get_error_code(resp: Response) -> str:
    """Extract machine-readable error code from API error response."""
    try:
        body = resp.json()
        return body.get("error", "") or ""
    except Exception:
        return ""


def _get_error_message(resp: Response) -> str:
    """Extract human-readable message from API error response."""
    try:
        body = resp.json()
        return body.get("message", "") or resp.text or resp.reason
    except Exception:
        return resp.reason or resp.text
