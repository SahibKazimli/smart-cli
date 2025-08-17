from langchain_google_vertexai import ChatVertexAI  
from langchain.prompts import ChatPromptTemplate       
from langchain_core.output_parsers import StrOutputParser  
from langchain_community.vectorstores import Redis
from typing import List, Dict
from dotenv import load_dotenv
from ..agent import retrieveChunks
import os

load_dotenv() 
credentialsPath = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = credentialsPath
print(os.getenv("GOOGLE_APPLICATION_CREDENTIALS"))  


def codeReview(query: str, folderName: str) -> Dict: 
    # Retrieves relevant code chunks from redis, and accepts a query
    chunks: List[Dict] = retrieveChunks(query, folderName, k=5)
    return {"query": query, "retrievedChunks":chunks}

if __name__ == "__main__":
    result = codeReview("Review build_index.py", "smart-cli")
    print(result)