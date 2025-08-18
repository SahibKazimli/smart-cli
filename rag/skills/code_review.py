from langchain_redis import RedisVectorStore
from typing import List, Dict
from dotenv import load_dotenv
import os

load_dotenv() 
credentialsPath = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = credentialsPath


def codeReview(query: str, indexName: str) -> Dict: 
    # Retrieves relevant code chunks from redis, and accepts a query
    from ..agent import retrieveChunks
    chunks: List[Dict] = retrieveChunks(query, indexName, k=5)
    return {"query": query, "retrievedChunks":chunks}

if __name__ == "__main__":
    result = codeReview("Review build_index.py", "._index")
    print(result)