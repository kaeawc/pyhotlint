class Timer:
    def sleep(self, n):
        return n


async def handler():
    t = Timer()
    # Local lookalike — `t.sleep` is not `time.sleep`, must not fire.
    t.sleep(1)
    return "ok"
