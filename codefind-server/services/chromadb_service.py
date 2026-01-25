import chromadb


class ChromaDBService:
    """Interface to ChromaDB vector store"""

    def __init__(self, host: str = "localhost", port: int = 8000):
        # Use HTTP client to connect to running ChromaDB server
        self.client = chromadb.HttpClient(host=host, port=port)

    def get_or_create_collection(self, collection_name: str):
        """Get existing collection or create new one"""
        try:
            collection = self.client.get_collection(name=collection_name)
        except Exception:
            # Collection doesn't exist, create it
            collection = self.client.create_collection(name=collection_name)

        return collection

    def health(self) -> bool:
        """Check if ChromaDB is accessible"""
        try:
            self.client.heartbeat()
            return True
        except Exception:
            return False
