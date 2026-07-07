"""Radiant Norma Python SDK.

Cliente Python oficial para a Radiant Norma API.

Uso::

    from radiant import Client

    c = Client(api_key="sk-...", if_id="00000")

    result = c.cadocs.validate("3040", xml_data)
    print(f"Valid: {result.valid}")
"""

from .client import (
    AuditService,
    CadocsService,
    Client,
    InsightsService,
    RadarService,
    SchemasService,
)
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

__all__ = [
    "Client",
    "CadocsService",
    "AuditService",
    "RadarService",
    "InsightsService",
    "SchemasService",
    "ValidationResult",
    "ValidationError",
    "CrossDocResult",
    "RuleDef",
    "ScanResult",
    "Change",
    "SchemaVersion",
    "LLMAnswer",
    "ErrorResponse",
]
