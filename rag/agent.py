from langchain_google_vertexai import ChatVertexAI
from langchain.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser
from langchain_google_vertexai import VertexAIEmbeddings
from langchain_redis import RedisVectorStore
from .skills.code_review import codeReview 
from vertexai import init
from typing import List, Dict
from dotenv import load_dotenv
import os

# Loading in credentials and necessary ID's
load_dotenv()
credentialsPath = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = credentialsPath
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
projectID = os.getenv("GCP_PROJECT_ID", "my-default-project")


init(
    project=projectID,
    location="us-central1"
)

modelName = "gemini-2.5-pro"
embeddingModel = VertexAIEmbeddings(model_name="text-embedding-004")


instructLLM = ChatVertexAI(
    model_name=modelName,
    temperature=0.2,
    max_output_tokens=3500
)

promptTemplate = """
<task>
  <description>
    You are an AI assistant integrated into a command-line interface.
    Explain things in plain text, without Markdown or special formatting.
    Your job is to:
    - Review code snippets and highlight potential errors.
    - Explain errors in clear, concise language.
    - Suggest fixes or improvements.
    - Optionally provide system or command-line suggestions.
  </description>
  <response_format>
    - Keep explanations short and actionable.
    - Avoid unnecessary verbosity.
    - Use plain text or simple structured output.
  </response_format>
  <constraints>
    - You may use available tokens to understand context.
    - But the response must be 500 tokens maximum.
    - Be precise and relevant to the input provided.
  </constraints>
</task>

<context>
{context}
</context>
User question: {query}
"""


chatPrompt = ChatPromptTemplate.from_template(promptTemplate)

def getVectorStore(indexName:str):
    return RedisVectorStore(redis_url=REDIS_URL, index_name=indexName, embeddings=embeddingModel)


def retrieveChunks(query:str, indexName:str, k=5) -> List[Dict]:
    # Retrieve the relevant chunks (k nearest neighbours)
    vectorStore = getVectorStore(indexName)

    # similarity_search returns List of Document objects
    chunks = vectorStore.similarity_search(query, k=k)

    # Format the output as a list of dictionaries
    return [{"text": chunk.page_content, "metadata": chunk.metadata} for chunk in chunks]



def generateResponse(query: str, indexName: str, k=5):
    chunks = retrieveChunks(query, indexName, k=k)
    context = "\n".join([chunk["text"] for chunk in chunks])

    # Fill the prompt template
    fullPrompt = chatPrompt.format(context=context, query=query)

    # Call the LLM
    response = instructLLM.invoke(fullPrompt)
    return response.content


def streamResponse(query: str, indexName: str, k=5):
    # Retrieve relevant chunks
    chunks = retrieveChunks(query, indexName, k=k)
    context = "\n".join([chunk["text"] for chunk in chunks])
    
    # Fill the prompt template
    fullPrompt = chatPrompt.format(context=context, query=query)

    # Stream response token by token
    print("\n--- LLM Streaming Response ---\n")
    for token in instructLLM.stream(fullPrompt):
        print(token.content, end="", flush=True)
    print("\n")  
    
    


if __name__ == "__main__":
    query = "What does build_index.py do?"
    print(query)
    cwd = os.getcwd()
    folder_name = os.path.basename(cwd)
    indexName = f"{folder_name}_index"

    # Stream the LLM output instead of waiting for the full response
    streamResponse(query, indexName)



