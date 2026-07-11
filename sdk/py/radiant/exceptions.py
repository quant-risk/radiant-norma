"""
Custom exceptions for Radiant Norma SDK.
"""


class HTTPError(Exception):
    """
    Raised for any non-2xx HTTP response.
    Attributes:
        status_code: HTTP status code.
        code: Machine-readable error code from API (e.g. "INVALID_CADOC").
        message: Human-readable error message.
    """

    def __init__(self, status_code: int, code: str = "", message: str = ""):
        self.status_code = status_code
        self.code = code
        self.message = message
        super().__init__(f"{code}: {message}" if code else f"HTTP {status_code}: {message}")

    def __repr__(self) -> str:
        return f"HTTPError(status_code={self.status_code!r}, code={self.code!r}, message={self.message!r})"


class ValidationError(Exception):
    """
    Raised when API returns a validation error response.
    Contains the list of individual field-level errors.
    """

    def __init__(self, errors: list):
        self.errors = errors
        super().__init__(f"Validation failed with {len(errors)} error(s)")
