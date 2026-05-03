import time


async def handler():
    # Suppressing a different rule must not hide this finding.
    time.sleep(1)  # pyhotlint: ignore[some-other-rule]
    return "ok"
