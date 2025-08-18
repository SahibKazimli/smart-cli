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
    # Dynamically derive index name based on current working directory
    cwd = os.getcwd()
    folder_name = os.path.basename(cwd)
    indexName = f"{folder_name}_index"

    result = codeReview("Review build_index.py", indexName)
    print(result)