from typing import Dict
from ..agent import retrieveChunks, streamResponse
from dotenv import load_dotenv
import os 

load_dotenv()
credentialsPath = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = credentialsPath


def explainError(errorText: str, indexName: str) -> Dict:
    """
    Given an error message or stack trace, retrieve relevant code snippets
    and provide an explanation using the LLM.
    """
    chunks = retrieveChunks(errorText, indexName, k=5)
    
    # Combine retrieved chunks as context
    context = "\n".join([chunk["text"] for chunk in chunks])
    
    # Craft a prompt for the LLM
    fullPrompt = (
        f"You are an AI assistant. A developer encountered the following error:\n\n"
        f"{errorText}\n\n"
        f"Relevant code snippets:\n{context}\n\n"
        "Explain the error in plain language and suggest possible fixes."
    )
    
    # Call LLM for explanation
    response = streamResponse(fullPrompt, indexName)
    
    return {"error": errorText, "retrievedChunks": chunks, "explanation": response}

