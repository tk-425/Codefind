import requests
from typing import List

class OllamaService:
  """Interface to Ollama embedding service"""

  def __init__(self, base_url: str = "http://localhost:11434"):
    self.base_url = base_url


  def embed(self, model: str, texts: List[str]) -> List[List[float]]:
    """Generate embeddings for texts using Ollama native API (batched)"""

    # Send all texts at once to Ollama for batch embedding
    response = requests.post(
      f"{self.base_url}/api/embed",
      json={"model": model, "input": texts},
      timeout=180  # 3 minute timeout for the entire batch
    )
    if response.status_code != 200:
      raise Exception(f"Ollama error ({response.status_code}): {response.text}")

    data = response.json()

    # Ollama returns "embeddings" field (list of embedding vectors, one per input text)
    return data["embeddings"]
  

  def health(self) -> bool:
    """Check if Ollama is accessible"""
    try:
      response = requests.get(f"{self.base_url}/api/tags", timeout=5)
      return response.status_code == 200
    except requests.RequestException:
      return False