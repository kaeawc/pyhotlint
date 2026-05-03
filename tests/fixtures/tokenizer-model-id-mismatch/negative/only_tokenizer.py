from transformers import AutoTokenizer


def load_tokenizer():
    return AutoTokenizer.from_pretrained("bert-base-uncased")
