import os
import json
import uuid
import time
import logging
import importlib.util
from typing import Dict, Any
from concurrent.futures import ThreadPoolExecutor, TimeoutError

DEFAULT_TIMEOUT = int(os.getenv("VALIDATION_TIMEOUT_SEC", "3"))

MODULE_REGISTRY = {
    "validator": "validate_share",
    "fraud": "detect_fraud"
}

class ValidationRunner:
    def __init__(self, timeout: int = DEFAULT_TIMEOUT):
        self.timeout = timeout
        self.executor = ThreadPoolExecutor(max_workers=4)
        self.modules = {}
        self._bootstrap_modules()

    def _bootstrap_modules(self):
        for key, name in MODULE_REGISTRY.items():
            path = os.path.join(os.path.dirname(__file__), f"{name}.py")
            spec = importlib.util.spec_from_file_location(name, path)
            if not spec or not spec.loader:
                raise RuntimeError(f"[BOOTSTRAP] Failed to load module: {name}")
            mod = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(mod)
            self.modules[key] = mod

    def _execute(self, func, payload: Dict[str, Any], label: str) -> Any:
        future = self.executor.submit(func, payload)
        try:
            result = future.result(timeout=self.timeout)
            return result
        except TimeoutError:
            raise RuntimeError(f"[TIMEOUT] Module '{label}' exceeded {self.timeout}s")
        except Exception as ex:
            raise RuntimeError(f"[RUNTIME] Module '{label}' failed: {ex}")

    def validate(self, share: Dict[str, Any]) -> Dict[str, Any]:
        ctx = {
            "id": str(uuid.uuid4()),
            "ts_start": time.time(),
            "status": "unknown",
            "reason": None,
            "basic_valid": None,
            "fraud_detected": None,
            "timing": {}
        }

        # Validator
        try:
            t0 = time.time()
            ctx["basic_valid"] = self._execute(self.modules["validator"].validate_share, share, "validator")
            ctx["timing"]["validation_ms"] = round((time.time() - t0) * 1000)
        except Exception as err:
            ctx.update({
                "status": "error",
                "reason": str(err),
                "duration_ms": round((time.time() - ctx["ts_start"]) * 1000)
            })
            self._log(ctx, level=logging.ERROR)
            return self._finalize(ctx)

        # Fraud Detection
        try:
            t1 = time.time()
            ctx["fraud_detected"] = self._execute(self.modules["fraud"].detect_fraud, share, "fraud")
            ctx["timing"]["fraud_check_ms"] = round((time.time() - t1) * 1000)
        except Exception as err:
            ctx.update({
                "status": "error",
                "reason": str(err),
                "duration_ms": round((time.time() - ctx["ts_start"]) * 1000)
            })
            self._log(ctx, level=logging.ERROR)
            return self._finalize(ctx)

        # Final decision
        ctx["status"] = "accepted" if ctx["basic_valid"] and not ctx["fraud_detected"] else "rejected"
        ctx["duration_ms"] = round((time.time() - ctx["ts_start"]) * 1000)
        self._log(ctx, level=logging.INFO)
        return self._finalize(ctx)

    def _log(self, context: Dict[str, Any], level=logging.INFO):
        log_payload = {
            "id": context["id"],
            "status": context["status"],
            "duration_ms": context.get("duration_ms"),
            "validation_ms": context["timing"].get("validation_ms"),
            "fraud_check_ms": context["timing"].get("fraud_check_ms"),
            "basic_valid": context.get("basic_valid"),
            "fraud_detected": context.get("fraud_detected"),
            "error": context.get("reason")
        }
        logging.log(level, f"[VALIDATION][{log_payload['id']}] Result: {json.dumps(log_payload)}")

    def _finalize(self, context: Dict[str, Any]) -> Dict[str, Any]:
        return {
            "id": context["id"],
            "status": context["status"],
            "reason": context["reason"],
            "basic_valid": context["basic_valid"],
            "fraud_detected": context["fraud_detected"],
            "timing": context["timing"],
            "duration_ms": context["duration_ms"]
        }

# Interface pública
def run_validation(input_data: Dict[str, Any]) -> Dict[str, Any]:
    engine = ValidationRunner()
    return engine.validate(input_data)
