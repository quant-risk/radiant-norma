"""
Smoke tests for Radiant Norma Python SDK.
Uses responses library to mock HTTP without hitting the network.
"""
from __future__ import annotations

import json
from datetime import datetime
from typing import Any, Dict
from unittest.mock import patch

import pytest
import responses

from radiant import Client
from radiant.exceptions import HTTPError
from radiant.models import (
    BatchGenerateResponse,
    ComplexityScore,
    CrossDocRulesResponse,
    GenerateResponse,
    HealthResponse,
    L4Comparison,
    RadarAlertsResponse,
    SchemaListResponse,
    ValidateResponse,
)


BASE = "https://api.radiantrisk.com/v1"


# ---------------------------------------------------------------------------
# Health
# ---------------------------------------------------------------------------

@responses.activate
def test_healthz() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/healthz",
        json={"status": "ok", "version": "3.36.2"},
        status=200,
    )
    c = Client(base_url=BASE, auth_token="test-token")
    resp = c.healthz()
    assert resp.status == "ok"
    assert resp.version == "3.36.2"
    assert len(responses.calls) == 1
    assert responses.calls[0].request.headers["Authorization"] == "Bearer test-token"


@responses.activate
def test_readyz() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/readyz",
        json={"status": "ready", "db": "ok"},
        status=200,
    )
    c = Client(base_url=BASE, if_id="if_demo")
    resp = c.readyz()
    assert resp.status == "ready"
    assert resp.db == "ok"
    assert responses.calls[0].request.headers["X-IF-ID"] == "if_demo"


# ---------------------------------------------------------------------------
# Schema
# ---------------------------------------------------------------------------

@responses.activate
def test_list_schemas_v2() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/schema",
        json={
            "schemas": [
                {
                    "cadoc_code": "3040",
                    "latest_version": "3.2",
                    "supported_versions": ["3.0", "3.1", "3.2"],
                    "field_count": 145,
                    "complexity": {"score": 0.72, "num_operacoes": 1540, "num_participantes": 230},
                },
                {
                    "cadoc_code": "4111",
                    "latest_version": "3.10",
                    "supported_versions": ["3.8", "3.9", "3.10"],
                    "field_count": 89,
                    "complexity": {"score": 0.55, "num_operacoes": 320, "num_participantes": 50},
                },
            ],
            "total": 2,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_schemas_v2()
    assert resp.total == 2
    assert resp.schemas[0].cadoc_code == "3040"
    assert resp.schemas[0].latest_version == "3.2"
    # Verify "3.10" > "3.9" is correctly returned (not "3.9")
    assert resp.schemas[1].cadoc_code == "4111"
    assert resp.schemas[1].latest_version == "3.10"
    assert resp.schemas[1].complexity is not None
    assert resp.schemas[1].complexity.score == 0.55


@responses.activate
def test_get_schema() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/schemas/3040",
        json={"cadoc": "3040", "version": "3.2", "effective_from": "2026-01-01"},
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.get_schema("3040")
    assert resp.cadoc == "3040"
    assert resp.version == "3.2"


@responses.activate
def test_list_versions() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/schemas/3040/versions",
        json={"cadoc": "3040", "versions": ["3.0", "3.1", "3.2"], "total": 3},
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_versions("3040")
    assert resp.versions == ["3.0", "3.1", "3.2"]
    assert resp.total == 3


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

@responses.activate
def test_validate_success() -> None:
    responses.add(
        responses.POST,
        f"{BASE}/validate",
        json={
            "cadoc_code": "3040",
            "data_base": "2026-06-30",
            "xml_hash": "abc123",
            "passed": True,
            "errors": [],
            "warnings": [],
            "duration_ms": 42,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.validate(cadoc="3040", xml="<SCRDocumento>...</SCRDocumento>")
    assert resp.passed is True
    assert len(resp.errors) == 0
    assert resp.duration_ms == 42


@responses.activate
def test_validate_with_errors() -> None:
    responses.add(
        responses.POST,
        f"{BASE}/validate",
        json={
            "cadoc_code": "3040",
            "passed": False,
            "errors": [
                {"code": "B12", "severity": "error", "field": "/SCR/Modalidade[1]", "message": "Valor deve ser maior que zero"},
            ],
            "warnings": [],
            "duration_ms": 18,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.validate(cadoc="3040", xml="<bad/>")
    assert resp.passed is False
    assert len(resp.errors) == 1
    assert resp.errors[0].code == "B12"


# ---------------------------------------------------------------------------
# Generation
# ---------------------------------------------------------------------------

@responses.activate
def test_generate_cadoc() -> None:
    responses.add(
        responses.POST,
        f"{BASE}/generate/3040",
        json={
            "cadoc_code": "3040",
            "data_base": "2026-06-30T00:00:00Z",
            "status": "generated",
            "generated": {
                "xml": "<SCRDocumento>...generated...</SCRDocumento>",
                "xml_hash": "abc123",
            },
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.generate_cadoc(
        cadoc="3040",
        request={
            "cadoc_code": "3040",
            "if_id": "if_demo",
            "cnpj": "12.345.678/0001-90",
            "data_base": "2026-06-30T00:00:00Z",
            "participantes": [
                {"id": "P001", "tipo": "PF", "nome": "Joao Silva", "cpf": "123.456.789-00", "rating": "AA"}
            ],
        },
    )
    assert resp.status == "generated"
    assert resp.generated is not None
    assert "generated" in resp.generated.xml


@responses.activate
def test_list_generate_fields() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/generate/3040/fields",
        json={
            "cadoc_code": "3040",
            "fields": [
                {"name": "CNPJ", "type": "string", "required": True},
                {"name": "DataBase", "type": "date", "required": True},
            ],
            "versions": ["3.0", "3.1", "3.2"],
            "complexity": {"score": 0.72, "num_operacoes": 1540, "num_participantes": 230},
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_generate_fields("3040")
    assert resp.cadoc_code == "3040"
    assert len(resp.fields) == 2
    assert resp.fields[0].required is True


@responses.activate
def test_list_source_adapters() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/generate/adapters",
        json={
            "adapters": [
                {"type": "manual", "name": "Manual Entry"},
                {"type": "api", "name": "REST API Connector"},
            ]
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_source_adapters()
    assert len(resp.adapters) == 2
    assert resp.adapters[0].type == "manual"


@responses.activate
def test_list_generate_history() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/generate/history",
        json={
            "items": [
                {
                    "id": "E",
                    "cadoc_code": "3040",
                    "data_base": "2026-06-30",
                    "generated_at": "2026-06-28T10:00:00Z",
                    "status": "accepted",
                    "passed": True,
                },
            ],
            "page": 1,
            "per_page": 20,
            "total": 1,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_generate_history(page=1, per_page=20)
    assert resp.total == 1
    assert resp.items[0].status == "accepted"
    assert resp.items[0].passed is True


# ---------------------------------------------------------------------------
# Cross-Doc
# ---------------------------------------------------------------------------

@responses.activate
def test_list_crossdoc_rules() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/crossdoc/rules",
        json={
            "rules": [
                {"code": "XD-4111-3040", "description": "XDRR01IPOC must match 3040", "severity": "error"},
            ],
            "total": 1,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_crossdoc_rules()
    assert resp.total == 1
    assert resp.rules[0].code == "XD-4111-3040"


@responses.activate
def test_crossdoc_validate() -> None:
    responses.add(
        responses.POST,
        f"{BASE}/crossdoc/validate",
        json={
            "passed": True,
            "errors": [],
            "warnings": [],
            "rules_executed": 3,
            "duration_ms": 95,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.crossdoc_validate(
        documents=[
            {"cadoc_code": "4111", "xml": "<XDRR01IPOC>...</XDRR01IPOC>"},
            {"cadoc_code": "3040", "xml": "<SCRDocumento>...</SCRDocumento>"},
        ]
    )
    assert resp.passed is True
    assert resp.rules_executed == 3


# ---------------------------------------------------------------------------
# Radar
# ---------------------------------------------------------------------------

@responses.activate
def test_list_radar_alerts() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/radar/alerts",
        json={
            "alerts": [
                {
                    "id": "alert-1",
                    "cadoc": "3040",
                    "change_type": "xsd_change",
                    "description": "Campo novo detectado no XSD",
                    "status": "open",
                    "created_at": "2026-06-01T00:00:00Z",
                }
            ],
            "total": 1,
            "page": 1,
            "per_page": 20,
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.list_radar_alerts(status="open")
    assert resp.total == 1
    assert resp.alerts[0].status == "open"


# ---------------------------------------------------------------------------
# L4
# ---------------------------------------------------------------------------

@responses.activate
def test_l4_compare() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/l4/compare",
        json={
            "current": {"id": "uuid-new", "cadoc_code": "3040", "passed": True},
            "previous": {"id": "uuid-old", "cadoc_code": "3040", "passed": False},
            "new_failures": [],
            "fixed_rules": [{"code": "B12", "severity": "error", "message": "Corrigido"}],
            "changed_fields": [],
            "alerts": [],
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.l4_compare(envio_id="abc-123")
    assert resp.current is not None
    assert resp.current.passed is True
    assert resp.previous.passed is False
    assert len(resp.fixed_rules) == 1
    assert resp.fixed_rules[0].code == "B12"


# ---------------------------------------------------------------------------
# Error handling
# ---------------------------------------------------------------------------

@responses.activate
def test_http_error() -> None:
    responses.add(
        responses.GET,
        f"{BASE}/schemas/9999",
        json={"error": "NOT_FOUND", "message": "schema not found"},
        status=404,
    )
    c = Client(base_url=BASE)
    with pytest.raises(HTTPError) as exc_info:
        c.get_schema("9999")
    assert exc_info.value.status_code == 404
    assert exc_info.value.code == "NOT_FOUND"
    assert "schema not found" in exc_info.value.message


@responses.activate
def test_http_error_no_body() -> None:
    responses.add(responses.GET, f"{BASE}/healthz", status=500, body="Internal Server Error")
    c = Client(base_url=BASE)
    with pytest.raises(HTTPError) as exc_info:
        c.healthz()
    assert exc_info.value.status_code == 500


# ---------------------------------------------------------------------------
# Context manager
# ---------------------------------------------------------------------------

@responses.activate
def test_context_manager() -> None:
    responses.add(responses.GET, f"{BASE}/healthz", json={"status": "ok", "version": "3.36.2"}, status=200)
    with Client(base_url=BASE) as c:
        resp = c.healthz()
        assert resp.status == "ok"
    # session should be closed after exiting context


# ---------------------------------------------------------------------------
# Auth header priority
# ---------------------------------------------------------------------------

@responses.activate
def test_bearer_token_preferred_over_ifid() -> None:
    responses.add(responses.GET, f"{BASE}/healthz", json={"status": "ok", "version": "3.36.2"}, status=200)
    c = Client(base_url=BASE, auth_token="jwt-token", if_id="if-fallback")
    c.healthz()
    req = responses.calls[0].request
    assert req.headers["Authorization"] == "Bearer jwt-token"
    assert req.headers["X-IF-ID"] == "if-fallback"


# ---------------------------------------------------------------------------
# Batch generate
# ---------------------------------------------------------------------------

@responses.activate
def test_generate_batch() -> None:
    responses.add(
        responses.POST,
        f"{BASE}/generate/batch",
        json={
            "results": [
                {"cadoc_code": "3040", "status": "generated", "xml_hash": "hash3040"},
                {"cadoc_code": "3050", "status": "generated", "xml_hash": "hash3050"},
            ],
            "passed": True,
            "message": "2/2 generated",
        },
        status=200,
    )
    c = Client(base_url=BASE)
    resp = c.generate_batch(
        request={
            "cadocs": [
                {"cadoc_code": "3040", "if_id": "if_demo"},
                {"cadoc_code": "3050", "if_id": "if_demo"},
            ],
            "run_crossdoc": True,
        }
    )
    assert resp.passed is True
    assert len(resp.results) == 2


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
