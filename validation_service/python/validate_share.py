# validation-service/python/validate_share.py

import time
from typing import Dict, Any, Tuple


class ShareValidator:
    def __init__(self, difficulty_threshold: int = 4, max_time_drift: int = 120):
        self.difficulty_threshold = difficulty_threshold
        self.max_time_drift = max_time_drift

    def run(self, share: Dict[str, Any]) -> Tuple[bool, str]:
        if not isinstance(share, dict):
            return False, "Expected input of type dict."

        if not self._fields_are_valid(share):
            return False, "Missing or invalid fields."

        if not self._timestamp_is_valid(share.get("timestamp", 0.0)):
            return False, "Timestamp is outside acceptable drift window."

        if not self._hash_meets_difficulty(share.get("hash", "")):
            return False, f"Hash does not meet required difficulty threshold of {self.difficulty_threshold}."

        return True, "Share is valid."

    def _fields_are_valid(self, share: Dict[str, Any]) -> bool:
        required_fields = {"worker_id", "hash", "nonce", "timestamp"}
        for key in required_fields:
            if key not in share:
                return False
            if not isinstance(share[key], (str, float, int)):
                return False
        return True

    def _timestamp_is_valid(self, ts: float) -> bool:
        if not isinstance(ts, (float, int)):
            return False
        current_ts = time.time()
        return abs(current_ts - ts) <= self.max_time_drift

    def _hash_meets_difficulty(self, hash_value: str) -> bool:
        if not isinstance(hash_value, str):
            return False
        return hash_value.startswith("0" * self.difficulty_threshold)


# Interface simplificada para chamadas externas
def validate_share(share: Dict[str, Any], difficulty: int = 4) -> Tuple[bool, str]:
    validator = ShareValidator(difficulty_threshold=difficulty)
    return validator.run(share)
