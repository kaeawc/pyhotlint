import json


def load_config(path):
    with open(path) as f:
        return json.load(f)


def parse(blob):
    return json.loads(blob)
