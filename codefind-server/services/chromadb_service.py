from typing import List
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

    def query(
        self,
        collection_name: str,
        query_embedding: List[float],
        top_k: int = 10,
        where: dict = None,
    ) -> dict:
        """Query a collection for similar chunks.

        Args:
            collection_name: Name of collection to query
            query_embedding: Query vector from Ollama
            top_k: Number of results to return
            where: Optional metadata filters for ChromaDB (e.g., {"language": "python"})

        Returns:
            dict with:
                - ids: List of chunk IDs
                - distances: List of L2 distances (0 = identical)
                - metadatas: List of metadata dicts
                - documents: List of chunk content

        Example where clause:
            {"language": {"$eq": "python"}}  # ChromaDB filter syntax
            {"file_path": {"$contains": "src/"}}
        """
        try:
            collection = self.client.get_collection(name=collection_name)
            results = collection.query(
                query_embeddings=[query_embedding],
                n_results=top_k,
                where=where,
                include=["distances", "documents", "metadatas"],
            )

            return {
                "ids": results["ids"][0] if results["ids"] else [],
                "distances": results["distances"][0] if results["distances"] else [],
                "documents": results["documents"][0] if results["documents"] else [],
                "metadatas": results["metadatas"][0] if results["metadatas"] else [],
            }
        except Exception as e:
            raise Exception(f"ChromaDB query error : {e}")

    def list_collections(self) -> List[str]:
        """List all available collections (projects)."""
        try:
            collections = self.client.list_collections()
            return [c.name for c in collections]
        except Exception as e:
            raise Exception(f"Failed to list collections: {e}")

    def collection_count(self, collection_name: str) -> int:
        """Get number of documents in a collection."""
        try:
            collection = self.client.get_collection(name=collection_name)
            return collection.count()
        except Exception as e:
            raise Exception(f"Failed to count collection: {e}")
