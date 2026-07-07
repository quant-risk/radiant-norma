"""Testes unitários do SDK Python."""

import pytest
import responses


@pytest.fixture
def client():
    """Cliente de teste apontando para o server mock."""
    from radiant import Client

    return Client(
        api_key="test-key",
        if_id="12345",
        base_url="https://api.test.local",
    )


@responses.activate
def test_validate_success(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/validate/3040",
        json={"valid": True, "errors": [], "warnings": []},
        status=200,
    )
    result = client.cadocs.validate("3040", b"<Doc3040/>")
    assert result.valid is True
    assert len(result.errors) == 0


@responses.activate
def test_validate_with_errors(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/validate/3040",
        json={
            "valid": False,
            "errors": [{"code": "F01", "severity": "E", "message": "data base inválida"}],
            "warnings": [],
        },
        status=200,
    )
    result = client.cadocs.validate("3040", b"<Doc3040/>")
    assert result.valid is False
    assert len(result.errors) == 1
    assert result.errors[0].code == "F01"


@responses.activate
def test_validate_api_error(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/validate/3040",
        json={"error": "bad_request", "message": "invalid xml"},
        status=400,
    )
    from radiant import ErrorResponse

    with pytest.raises(ErrorResponse) as exc_info:
        client.cadocs.validate("3040", b"bad")
    assert exc_info.value.error == "bad_request"


@responses.activate
def test_list_rules(client):
    responses.add(
        responses.GET,
        "https://api.test.local/v1/rules/3040",
        json={
            "rules": [
                {"code": "F01", "severity": "E", "message": "data base inválida"},
            ]
        },
        status=200,
    )
    rules = client.audit.list_rules("3040")
    assert len(rules) == 1
    assert rules[0].code == "F01"


@responses.activate
def test_radar_scan(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/radar/scan",
        json={"id": "scan-1", "if_id": "12345", "cadoc": "3040", "status": "clean", "changes": []},
        status=200,
    )
    scan = client.radar.scan("3040")
    assert scan.status == "clean"


@responses.activate
def test_insights_ask(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/insights/ask",
        json={"answer": "resposta teste", "model": "minimax"},
        status=200,
    )
    ans = client.insights.ask("o que é PLM?")
    assert ans.answer == "resposta teste"


@responses.activate
def test_schemas_list_versions(client):
    responses.add(
        responses.GET,
        "https://api.test.local/v1/schemas/3040/versions",
        json={"versions": [{"id": 1, "cadoc_code": "3040", "effective_from": "2026-01"}]},
        status=200,
    )
    versions = client.schemas.list_versions("3040")
    assert len(versions) == 1
    assert versions[0].cadoc_code == "3040"


@responses.activate
def test_validate_cross_doc(client):
    responses.add(
        responses.POST,
        "https://api.test.local/v1/crossdoc/validate",
        json={
            "passed": True,
            "rules_run": ["XD4111CNPJConsistente"],
            "rules_skipped": [],
            "duration_ms": 120,
            "errors": [],
            "warnings": [],
        },
        status=200,
    )
    result = client.cadocs.validate_cross_doc({"3040": b"<Doc3040/>", "4111": b"<Doc4111/>"})
    assert result.passed is True
    assert "XD4111CNPJConsistente" in result.rules_run


def test_client_default_base_url():
    from radiant import Client

    c = Client(api_key="test")
    assert c.base_url == "https://api.radiantnorma.com"


def test_validation_error_fields():
    from radiant import ValidationError

    e = ValidationError(code="F01", severity="E", field="dataBase", message="inválida")
    assert e.code == "F01"
    assert e.severity == "E"
    assert e.field == "dataBase"
