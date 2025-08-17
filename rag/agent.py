from langchain_google_vertexai import ChatVertexAI
from langchain.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser
from langchain_google_vertexai import VertexAIEmbeddings
from langchain_community.vectorstores import Redis
from skills.code_review import codeReview
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
    location="us-central1" # Ensure this matches your Vertex AI setup
)

modelName = "gemini-2.5-pro"
embeddingModel = VertexAIEmbeddings(model_name="text-embedding-004")


instructLLM = ChatVertexAI(
    model_name=modelName,
    temperature=0.2,
    max_output_tokens=400
)


promptTemplate = """
<task>
  <description>
    You are an AI assistant integrated into a command-line interface.
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
    - Maximum response: 300 tokens.
    - Be precise and relevant to the input provided.
  </constraints>
</task>"""



def getVectorStore(folderName:str):
    indexName = folderName + "_index"
    return Redis(redis_url=REDIS_URL, index_name=indexName, embedding=embeddingModel)


def retrieveChunks(query:str, folderName:str, k=5) -> List[Dict]:
    # Retrieve the relevant chunks (k nearest neighbours)
    vectorStore = getVectorStore(folderName)

    # similarity_search returns List of Document objects
    chunks = vectorStore.similarity_search(query, k=k)

    # Format the output as a list of dictionaries
    return [{"text": chunk.page_content, "metadata": chunk.metadata} for chunk in chunks]


if __name__ == "__main__":
    # Make sure you have ingested data for "smart-cli" folder before running this
    print("Retrieving code review chunks...")
    result = codeReview("Review build_index.py", "smart-cli")
    print(result)