import requests


async def fetch(url):
    resp = requests.get(url, timeout=5)
    return resp.json()
