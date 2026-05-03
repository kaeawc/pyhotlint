async def fetch_each(urls):
    out = []
    for url in urls:
        out.append(await fetch(url))
    return out


async def fetch(url):
    return url
