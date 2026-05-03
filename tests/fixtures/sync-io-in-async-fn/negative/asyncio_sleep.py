import asyncio


async def handler():
    await asyncio.sleep(0.5)
    return "ok"
