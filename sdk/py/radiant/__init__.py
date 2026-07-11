"""
Radiant Norma Python SDK
~~~~~~~~~~~~~~~~~~~~~~~~
Official Python client for Radiant Norma API.

Usage::

    from radiant import Client

    c = Client(
        base_url="https://api.radiantrisk.com/v1",
        auth_token="your-jwt-token",
    )

    # Health check
    health = c.healthz()

    # List schemas
    schemas = c.list_schemas_v2()

    # Generate a CADOC
    resp = c.generate_cadoc(
        cadoc="3040",
        request={
            "if_id": "if_demo",
            "cnpj": "12.345.678/0001-90",
            "data_base": "2026-06-30T00:00:00Z",
        },
    )

    # Validate generated XML
    result = c.validate(cadoc="3040", xml=xml_content)

Auto-generated from OpenAPI v3.36.2 (docs/openapi/v1.yaml).
"""
from radiant.client import Client
from radiant.exceptions import HTTPError
from radiant.models import (
    AdapterInfo,
    BatchGenerateRequest,
    BatchGenerateResponse,
    ComplexityScore,
    CrossDocInput,
    CrossDocValidateRequest,
    CrossDocValidateResponse,
    FieldsResponse,
    GenerateRequest,
    GenerateResponse,
    GenerationHistoryItem,
    GenerationHistoryResponse,
    HealthResponse,
    ReadyResponse,
    Rule,
    RuleListResponse,
    SchemaInfo,
    SchemaListResponse,
    SchemaVersion,
    SchemasResponse,
    ValidateResponse,
    VersionsResponse,
)

__all__ = [
    "Client",
    "HTTPError",
    "ValidationError",
    # models
    "AdapterInfo",
    "BatchGenerateRequest",
    "BatchGenerateResponse",
    "ComplexityScore",
    "CrossDocInput",
    "CrossDocValidateRequest",
    "CrossDocValidateResponse",
    "FieldsResponse",
    "GenerateRequest",
    "GenerateResponse",
    "GenerationHistoryItem",
    "GenerationHistoryResponse",
    "HealthResponse",
    "ReadyResponse",
    "Rule",
    "RuleListResponse",
    "SchemaInfo",
    "SchemaListResponse",
    "SchemaVersion",
    "SchemasResponse",
    "ValidationResponse",
    "VersionsResponse",
]
