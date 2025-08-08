import time
import json
import math
from typing import List, Dict, Tuple, Any, Optional
from collections import defaultdict
from pathlib import Path


class FraudPolicy:
    def __init__(self, name: str = "default", window_seconds: int = 300, max_shares: int = 200, min_avg_nonce: float = 1000.0):
        self.name = name
        self.window_seconds = window_seconds
        self.max_shares = max_shares
        self.min_avg_nonce = min_avg_nonce


class FraudDetector:
    def __init__(
        self,
        policy: FraudPolicy = FraudPolicy(),
        persist_path: Optional[str] = None,
        logger: Optional[callable] = None
    ):
        self.policy = policy
        self.log = logger or (lambda *a, **k: None)
        self.persist_path = Path(persist_path) if persist_path else None
        self.cache: Dict[str, List[Tuple[float, int]]] = defaultdict(list)

    def analyze(self, share: Dict[str, Any]) -> Tuple[bool, str]:
        key = self._compose_key(share)
        nonce = share.get("nonce")
        ts = share.get("timestamp")

        if not key or not isinstance(nonce, int) or not isinstance(ts, (float, int)):
            self.log("warn", "Malformed share ignored.", share)
            return True, "Malformed or incomplete share."

        self._record(key, nonce, ts)

        if self._is_spam(key):
            msg = "Excessive share frequency."
            self._report_suspicious(share, msg)
            return False, msg

        if self._nonce_pattern(key):
            msg = "Suspicious nonce uniformity."
            self._report_suspicious(share, msg)
            return False, msg

        return True, "Accepted."

    def _record(self, key: str, nonce: int, ts: float):
        self.cache[key].append((ts, nonce))
        min_ts = ts - self.policy.window_seconds
        self.cache[key] = [x for x in self.cache[key] if x[0] >= min_ts]

    def _is_spam(self, key: str) -> bool:
        return len(self.cache[key]) > self.policy.max_shares

    def _nonce_pattern(self, key: str) -> bool:
        nonces = [n for _, n in self.cache[key]]
        if not nonces:
            return False
        avg = sum(nonces) / len(nonces)
        var = sum((x - avg) ** 2 for x in nonces) / len(nonces)
        std = math.sqrt(var)
        return avg < self.policy.min_avg_nonce or std < 10.0

    def _compose_key(self, share: Dict[str, Any]) -> str:
        wid = str(share.get("worker_id", "")).strip()
        ip = str(share.get("ip", "")).strip()
        return f"{wid}|{ip}"

    def _report_suspicious(self, share: Dict[str, Any], reason: str):
        payload = {
            "reason": reason,
            "share": share,
            "ts": time.time()
        }
        self.log("alert", reason, share)
        if self.persist_path:
            try:
                with self.persist_path.open("a", encoding="utf-8") as f:
                    f.write(json.dumps(payload) + "\n")
            except Exception as err:
                self.log("error", "Failed to persist fraud report.", {"err": str(err)})


# External callable
def detect_fraud(share: Dict[str, Any], detector: Optional[FraudDetector] = None) -> Tuple[bool, str]:
    global_detector = detector or FraudDetector()
    return global_detector.analyze(share)
