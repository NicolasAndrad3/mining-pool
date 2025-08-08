"""
Módulo principal de inicialização do pacote de análise de shares.
Responsável por expor as funções críticas de validação e detecção.
"""

__version__ = "0.1.0"
__author__ = "Nicolas A. A."
__license__ = "MIT"
__all__: list[str] = [
    "validate_share",
    "detect_fraud",
    "runner",
]

# Registro de diagnóstico leve em modo debug
import logging
_logger = logging.getLogger(__name__)
_logger.debug("Módulo 'validation-service.python' carregado com sucesso.")

# Importação protegida para controle de integridade do pacote
try:
    from .validate_share import validate_share, ValidationResult
    from .detect_fraud import detect_fraud, FraudDetector, FraudPolicy
    from .runner import run_analysis_pipeline
except ImportError as err:
    raise ImportError(
        f"[Pacote: validation-service] Falha ao carregar dependência interna: {err}"
    )
