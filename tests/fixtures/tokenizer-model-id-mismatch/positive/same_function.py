from transformers import AutoTokenizer, AutoModel


def load():
    tokenizer = AutoTokenizer.from_pretrained("bert-base-uncased")
    model = AutoModel.from_pretrained("roberta-base")
    return tokenizer, model
