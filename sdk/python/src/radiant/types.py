"""Tipos compartilhados do SDK Radiant Norma."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional


@dataclass
class ValidationError:
    code: str
    severity: str  # E, A, I
    field: Optional[str] = None
    message: str = ""


@dataclass
class ValidationResult:
    valid: bool
    errors: List[ValidationError] = field(default_factory=list)
    warnings: List[ValidationError] = field(default_factory=list)


@dataclass
class CrossDocResult:
    passed: bool
    rules_run: List[str] = field(default_factory=list)
    rules_skipped: List[str] = field(default_factory=list)
    duration_ms: int = 0
    errors: List[ValidationError] = field(default_factory=list)
    warnings: List[ValidationError] = field(default_factory=list)


@dataclass
class ErrorResponse(Exception):
    error: str
    message: str = ""
    code: str = ""

    def __str__(self) -> str:
        return f"[{self.error}] {self.message}"


@dataclass
class RuleDef:
    code: str
    severity: str  # E, A, I
    message: str


@dataclass
class Envio:
    id: int
    if_id: str
    cadoc: str
    data_base: str
    status: str
    rules_passed: int = 0
    rules_failed: int = 0
    submitted_at: Optional[datetime] = None
    validated_at: Optional[datetime] = None


@dataclass
class Change:
    tag: str
    kind: str  # added, removed, modified
    attr: Optional[str] = None
    old_value: str = ""
    new_value: str = ""


@dataclass
class ScanResult:
    id: str
    if_id: str
    cadoc: str
    status: str  # clean, changed, error
    changes: List[Change] = field(default_factory=list)
    scanned_at: Optional[datetime] = None


@dataclass
class SchemaVersion:
    id: int
    cadoc_code: str
    effective_from: str
    source_uri: str = ""
    changelog: str = ""
    created_at: Optional[datetime] = None


@dataclass
class LLMAnswer:
    answer: str
    model: str = ""
