"""Cliente HTTP principal do SDK Radiant Norma."""

import json
from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple

import requests

from .types import (
    Change,
    CrossDocResult,
    ErrorResponse,
    LLMAnswer,
    RuleDef,
    ScanResult,
    SchemaVersion,
    ValidationError,
    ValidationResult,
)


class Client:
    """Cliente principal da Radiant Norma API."""

    def __init__(
        self,
        api_key: str,
        if_id: str = "",
        base_url: str = "https://api.radiantnorma.com",
        timeout: int = 30,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.if_id = if_id
        self.timeout = timeout

        self.cadocs: CadocsService = CadocsService(self)
        self.audit: AuditService = AuditService(self)
        self.radar: RadarService = RadarService(self)
        self.insights: InsightsService = InsightsService(self)
        self.schemas: SchemasService = SchemasService(self)

    def _request(
        self,
        method: str,
        path: str,
        body: Any = None,
        params: Optional[Dict] = None,
    ) -> dict:
        url = f"{self.base_url}{path}"
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        if self.if_id:
            headers["X-IF-ID"] = self.if_id

        response = requests.request(
            method,
            url,
            json=body,
            params=params,
            headers=headers,
            timeout=self.timeout,
        )

        if response.status_code >= 400:
            err_data = {}
            try:
                err_data = response.json()
            except Exception:
                pass
            raise ErrorResponse(
                error=err_data.get("error", "http_error"),
                message=err_data.get("message", response.text or response.reason),
                code=err_data.get("code", ""),
            )

        return response.json()

    def _get(self, path: str, params: Optional[dict] = None) -> dict:
        return self._request("GET", path, params=params)

    def _post(self, path: str, body: Any = None) -> dict:
        return self._request("POST", path, body=body)


class CadocsService:
    """Validação e envio de documentos CADOC."""

    def __init__(self, client: Client):
        self._c = client

    def validate(self, cadoc: str, xml_data: Any) -> ValidationResult:
        """Valida um documento CADOC (não envia ao BACEN).

        Args:
            cadoc: código do CADOC (ex: "3040", "4111")
            xml_data: conteúdo XML do documento

        Returns:
            ValidationResult com passed/errors/warnings
        """
        payload = {
            "cadoc_code": cadoc,
            "xml": xml_data if isinstance(xml_data, str) else xml_data.decode(),
        }
        data = self._c._post("/v1/validate", body=payload)
        return _parse_validation_result(data)

    def validate_cross_doc(self, docs: Dict[str, Any]) -> CrossDocResult:
        """Valida consistência entre múltiplos documentos.

        Args:
            docs: map cadoc_code -> xml_bytes

        Returns:
            CrossDocResult com resultados por regra
        """
        raw_docs = {}
        for k, v in docs.items():
            raw_docs[k] = v if isinstance(v, str) else v.decode()
        data = self._c._post("/v1/crossdoc/validate", body={"cadocs": raw_docs})
        return _parse_cross_doc_result(data)


class AuditService:
    """Regras de auditoria."""

    def __init__(self, client: Client):
        self._c = client

    def list_rules(self, cadoc: str) -> List[RuleDef]:
        """Lista todas as regras de auditoria para um CADOC."""
        data = self._c._get(f"/v1/rules/{cadoc}")
        return [RuleDef(**r) for r in data.get("rules", [])]


class RadarService:
    """Detecção de mudanças de layout."""

    def __init__(self, client: Client):
        self._c = client

    def scan(self, cadoc: str) -> ScanResult:
        """Executa um radar scan para um CADOC."""
        data = self._c._post(f"/v1/radar/scan?cadoc={cadoc}")
        return _parse_scan_result(data)


class InsightsService:
    """Insights LLM."""

    def __init__(self, client: Client):
        self._c = client

    def ask(self, question: str) -> LLMAnswer:
        """Faz uma pergunta em linguagem natural sobre o ambiente do tenant."""
        data = self._c._post("/v1/insights/ask", body={"question": question})
        return LLMAnswer(**data)


class SchemasService:
    """Schema registry."""

    def __init__(self, client: Client):
        self._c = client

    def list_versions(self, cadoc: str) -> List[SchemaVersion]:
        """Retorna histórico de versões de um CADOC."""
        data = self._c._get(f"/v1/schemas/{cadoc}/versions")
        return [SchemaVersion(**v) for v in data.get("versions", [])]

    def get_changelog(self, cadoc: str) -> List[SchemaVersion]:
        """Retorna timeline de changelogs de um CADOC."""
        data = self._c._get(f"/v1/schemas/{cadoc}/changelog")
        return [SchemaVersion(**e) for e in data.get("entries", [])]


# ─── helpers de parse ────────────────────────────────────────────────────────


def _parse_validation_result(data: dict) -> ValidationResult:
    return ValidationResult(
        passed=data.get("passed", False),
        data_base=data.get("data_base", ""),
        xml_hash=data.get("xml_hash", ""),
        errors=[ValidationError(**e) for e in data.get("errors", [])],
        warnings=[ValidationError(**w) for w in data.get("warnings", [])],
        duration_ms=data.get("duration_ms", 0),
    )


def _parse_cross_doc_result(data: dict) -> CrossDocResult:
    return CrossDocResult(
        passed=data.get("passed", False),
        errors=[ValidationError(**e) for e in data.get("errors", [])],
        warnings=[ValidationError(**w) for w in data.get("warnings", [])],
        rules_run=data.get("rules_run", []),
        rules_skipped=data.get("rules_skipped", []),
        duration_ms=data.get("duration_ms", 0),
    )


def _parse_scan_result(data: dict) -> ScanResult:
    return ScanResult(
        id=data.get("id", ""),
        if_id=data.get("if_id", ""),
        cadoc=data.get("cadoc", ""),
        status=data.get("status", ""),
        changes=[Change(**c) for c in data.get("changes", [])],
        scanned_at=_parse_datetime(data.get("scanned_at")),
    )


def _parse_datetime(value: Any) -> Any:
    if value is None:
        return None
    if isinstance(value, str):
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00"))
        except Exception:
            return value
    return value
